package operations

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io"
	"slices"
	"sort"
	"strings"
	"time"
)

const maxRecord = 64 << 20

type sequence struct {
	Table  string `json:"table"`
	Column string `json:"column"`
	Next   int64  `json:"next"`
}
type snapshotHeader struct {
	InstanceID string     `json:"instanceId"`
	Version    int        `json:"version"`
	SchemaHash string     `json:"schemaHash"`
	Tables     []string   `json:"tables"`
	Sequences  []sequence `json:"sequences"`
	CreatedAt  time.Time  `json:"createdAt"`
}
type record struct {
	Table string          `json:"table"`
	Row   json.RawMessage `json:"row"`
}

func catalog(ctx context.Context, tx pgx.Tx) (snapshotHeader, error) {
	h := snapshotHeader{Version: 1, Tables: []string{}, Sequences: []sequence{}, CreatedAt: time.Now().UTC()}
	if e := tx.QueryRow(ctx, "SELECT instance_id::text FROM ops_control WHERE singleton").Scan(&h.InstanceID); e != nil {
		return h, e
	}
	var migrationJSON []byte
	if e := tx.QueryRow(ctx, "SELECT json_agg(json_build_array(name,sha256) ORDER BY name)::text FROM schema_migrations").Scan(&migrationJSON); e != nil {
		return h, e
	}
	h.SchemaHash = hash(migrationJSON)
	rows, e := tx.Query(ctx, "SELECT tablename FROM pg_tables WHERE schemaname=current_schema() AND tablename<>'schema_migrations' ORDER BY tablename")
	if e != nil {
		return h, e
	}
	names := []string{}
	for rows.Next() {
		var name string
		if e = rows.Scan(&name); e != nil {
			rows.Close()
			return h, e
		}
		names = append(names, name)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return h, e
	}
	deps := map[string][]string{}
	rows, e = tx.Query(ctx, "SELECT c.relname,p.relname FROM pg_constraint fk JOIN pg_class c ON c.oid=fk.conrelid JOIN pg_class p ON p.oid=fk.confrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE fk.contype='f' AND n.nspname=current_schema() AND c.oid<>p.oid")
	if e != nil {
		return h, e
	}
	for rows.Next() {
		var c, p string
		if e = rows.Scan(&c, &p); e != nil {
			rows.Close()
			return h, e
		}
		deps[c] = append(deps[c], p)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return h, e
	}
	for len(names) > 0 {
		progress := false
		for _, name := range slices.Clone(names) {
			ready := true
			for _, dep := range deps[name] {
				if !slices.Contains(h.Tables, dep) {
					ready = false
					break
				}
			}
			if ready {
				h.Tables = append(h.Tables, name)
				names = slices.DeleteFunc(names, func(v string) bool { return v == name })
				progress = true
			}
		}
		if !progress {
			return h, ErrBackup
		}
	}
	rows, e = tx.Query(ctx, "SELECT table_name,column_name FROM information_schema.columns WHERE table_schema=current_schema() AND is_identity='YES' ORDER BY table_name,column_name")
	if e != nil {
		return h, e
	}
	for rows.Next() {
		var seq sequence
		if e = rows.Scan(&seq.Table, &seq.Column); e != nil {
			rows.Close()
			return h, e
		}
		h.Sequences = append(h.Sequences, seq)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return h, e
	}
	for i := range h.Sequences {
		seq := &h.Sequences[i]
		name, e := sequenceName(ctx, tx, seq.Table, seq.Column)
		if e != nil {
			return h, e
		}
		var value int64
		var called bool
		if e = tx.QueryRow(ctx, "SELECT last_value,is_called FROM "+name).Scan(&value, &called); e != nil {
			return h, e
		}
		if called {
			value++
		}
		if value < 1 {
			return h, ErrBackup
		}
		seq.Next = value
	}
	return h, nil
}
func sequenceName(ctx context.Context, tx pgx.Tx, table, column string) (string, error) {
	var name string
	e := tx.QueryRow(ctx, "SELECT quote_ident(n.nspname)||'.'||quote_ident(c.relname) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.oid=pg_get_serial_sequence(quote_ident(current_schema())||'.'||quote_ident($1),$2)::regclass", table, column).Scan(&name)
	return name, e
}
func (s Service) snapshot(ctx context.Context, w io.Writer) error {
	tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, "SET LOCAL idle_in_transaction_session_timeout='60s'"); e != nil {
		return e
	}
	h, e := catalog(ctx, tx)
	if e != nil {
		return e
	}
	enc := json.NewEncoder(w)
	if e = enc.Encode(h); e != nil {
		return e
	}
	for _, table := range h.Tables {
		rows, e := tx.Query(ctx, "SELECT row_to_json(t)::text FROM "+pgx.Identifier{table}.Sanitize()+" t")
		if e != nil {
			return e
		}
		for rows.Next() {
			var b json.RawMessage
			if e = rows.Scan(&b); e != nil {
				rows.Close()
				return e
			}
			if len(b) > maxRecord-1024 {
				rows.Close()
				return ErrBackup
			}
			if e = enc.Encode(record{Table: table, Row: b}); e != nil {
				rows.Close()
				return e
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
func scanSnapshot(r io.Reader, header func(snapshotHeader) error, each func(record) error) error {
	scan := bufio.NewScanner(r)
	scan.Buffer(make([]byte, 65536), maxRecord)
	if !scan.Scan() {
		return ErrBackup
	}
	var h snapshotHeader
	if e := decodeStrict(scan.Bytes(), &h); e != nil || h.Version != 1 || len(h.Tables) == 0 || len(h.SchemaHash) != 64 {
		return ErrBackup
	}
	names := map[string]int{}
	for i, t := range h.Tables {
		if _, ok := names[t]; ok {
			return ErrBackup
		}
		names[t] = i
	}
	if e := header(h); e != nil {
		return e
	}
	last := -1
	for scan.Scan() {
		var rec record
		if e := decodeStrict(scan.Bytes(), &rec); e != nil {
			return ErrBackup
		}
		index, ok := names[rec.Table]
		if !ok || index < last || len(rec.Row) < 2 || rec.Row[0] != '{' {
			return ErrBackup
		}
		last = index
		if each != nil {
			if e := each(rec); e != nil {
				return e
			}
		}
	}
	return scan.Err()
}
func decodeStrict(b []byte, v any) error {
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	if d.Decode(new(any)) != io.EOF {
		return ErrBackup
	}
	return nil
}
func sameSchema(a, b snapshotHeader) bool {
	return a.SchemaHash == b.SchemaHash && slices.Equal(a.Tables, b.Tables) && len(a.Sequences) == len(b.Sequences)
}
func (s Service) validateSnapshot(ctx context.Context, r io.Reader) error {
	tx, e := s.DB.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if e != nil {
		return e
	}
	h, e := catalog(ctx, tx)
	_ = tx.Rollback(context.Background())
	if e != nil {
		return e
	}
	return scanSnapshot(r, func(v snapshotHeader) error {
		if !sameSchema(h, v) {
			return ErrBackup
		}
		for i, seq := range v.Sequences {
			if seq.Table != h.Sequences[i].Table || seq.Column != h.Sequences[i].Column || seq.Next < 1 {
				return ErrBackup
			}
		}
		return nil
	}, nil)
}
func (s Service) importSnapshot(ctx context.Context, tx pgx.Tx, r io.Reader, recoveryKEK []byte) error {
	current, e := catalog(ctx, tx)
	if e != nil {
		return e
	}
	tables := []string{}
	for _, name := range current.Tables {
		if !strings.HasPrefix(name, "ops_") {
			tables = append(tables, name)
		}
	}
	quoted := []string{}
	for _, name := range tables {
		quoted = append(quoted, pgx.Identifier{name}.Sanitize())
	}
	if len(tables) == 0 {
		return ErrBackup
	}
	// User triggers enforce normal lifecycle rules, which are intentionally not
	// replayed during recovery. Foreign-key/check constraints remain enabled.
	for _, table := range quoted {
		if _, e = tx.Exec(ctx, "ALTER TABLE "+table+" DISABLE TRIGGER USER"); e != nil {
			return e
		}
	}
	if _, e = tx.Exec(ctx, "TRUNCATE "+strings.Join(quoted, ",")); e != nil {
		return e
	}
	var source snapshotHeader
	e = scanSnapshot(r, func(h snapshotHeader) error {
		if !sameSchema(current, h) {
			return ErrBackup
		}
		source = h
		return nil
	}, func(rec record) error {
		if strings.HasPrefix(rec.Table, "ops_") {
			return nil
		}
		table := pgx.Identifier{rec.Table}.Sanitize()
		_, e := tx.Exec(ctx, "INSERT INTO "+table+" OVERRIDING SYSTEM VALUE SELECT * FROM json_populate_record(NULL::"+table+",$1::json)", string(rec.Row))
		return e
	})
	if e != nil {
		return e
	}
	for _, table := range quoted {
		if _, e = tx.Exec(ctx, "ALTER TABLE "+table+" ENABLE TRIGGER USER"); e != nil {
			return e
		}
	}
	sort.Slice(source.Sequences, func(i, j int) bool { return source.Sequences[i].Table < source.Sequences[j].Table })
	for _, seq := range source.Sequences {
		if strings.HasPrefix(seq.Table, "ops_") {
			continue
		}
		name, e := sequenceName(ctx, tx, seq.Table, seq.Column)
		if e != nil {
			return e
		}
		var next int64
		if e = tx.QueryRow(ctx, "SELECT COALESCE(max("+pgx.Identifier{seq.Column}.Sanitize()+"),0)+1 FROM "+pgx.Identifier{seq.Table}.Sanitize()).Scan(&next); e != nil {
			return e
		}
		next = max(next, seq.Next)
		if next < 1 {
			return ErrBackup
		}
		if _, e = tx.Exec(ctx, fmt.Sprintf("ALTER SEQUENCE %s RESTART WITH %d", name, next)); e != nil {
			return e
		}
	}
	return rewrapApplication(ctx, tx, recoveryKEK, s.Keys.KEK)
}
