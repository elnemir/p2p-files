export interface paths {
  "/api/v1/status": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read aggregate service status
     * @description Aggregate database reachability, recovery gate and runtime generation status; no topology, history or component identifiers. Does not prove production readiness.
     */
    get: operations["getStatus"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/branding": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read public installation branding
     * @description Public nonsecret presentation only; no configuration secrets, actor or history. No-store. Revision zero uses built-in defaults. Palette text contrast and image budgets are validated on apply.
     */
    get: operations["getBranding"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/branding": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read the current branding revision
     * @description Read the current branding revision. ADMIN cookie session with reauthentication within five minutes is required.
     */
    get: operations["getAdminBranding"];
    /**
     * Apply a branding revision
     * @description Atomic compare-and-swap and audit. Requires current ADMIN cookie session, fresh reauthentication, Origin and CSRF. Invalid input does not change branding; concurrent/stale revisions return 409. Reset by applying built-in defaults as a new revision. Request limit 600 KiB. ADMIN cookie session with reauthentication within five minutes is required.
     */
    put: operations["putAdminBranding"];
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/license": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read offline license state and installation request identity
     * @description Read offline license state and installation request identity. ADMIN cookie session with reauthentication within five minutes is required.
     */
    get: operations["getAdminLicense"];
    /**
     * Apply a signed offline license with epoch CAS
     * @description ADMIN reauthentication, Origin and CSRF required. Signature, installation binding and monotonic entitlement revision are checked. Invalid replacement preserves active rights. Future licenses remain pending until notBefore. ADMIN cookie session with reauthentication within five minutes is required.
     */
    put: operations["putAdminLicense"];
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/license/preview": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Verify a candidate offline license without applying it
     * @description Verify a candidate offline license without applying it. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminLicensePreview"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/banner": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read the global announcement
     * @description Read the global announcement.
     */
    get: operations["getBanner"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/login": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Sign in with email and password
     * @description Requires exact publicOrigin. Sets HttpOnly SameSite=Strict filexSession cookie. Generic errors and durable per-IP/per-email throttling prevent enumeration. Client IP uses the socket peer unless a configured single Traefik hop supplies valid forwarded headers; network allowlist/proxy/HTTPS guards apply before login. Store csrfToken for unsafe requests.
     */
    post: operations["postAuthLogin"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/me": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read current principal
     * @description Read current principal.
     */
    get: operations["getAuthMe"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/logout": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * End the current session
     * @description End the current session.
     */
    post: operations["postAuthLogout"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/reauth": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Reauthenticate for five minutes
     * @description Reauthenticate for five minutes.
     */
    post: operations["postAuthReauth"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/password": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    /**
     * Change password and invalidate sessions and tokens
     * @description Change password and invalidate sessions and tokens.
     */
    patch: operations["patchAuthPassword"];
    trace?: never;
  };
  "/api/v1/auth/sessions": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List own sessions
     * @description List own sessions.
     */
    get: operations["getAuthSessions"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/sessions/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    /**
     * Revoke an owned session
     * @description Revoke an owned session.
     */
    delete: operations["deleteAuthSessionsId"];
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/history": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List own login history
     * @description List own login history.
     */
    get: operations["getAuthHistory"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/tokens": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List own API token metadata
     * @description List own API token metadata.
     */
    get: operations["getAuthTokens"];
    put?: never;
    /**
     * Create a scoped API token
     * @description Cookie session and fresh reauthentication required; token is returned once. No privileged ADMIN scopes may be minted.
     */
    post: operations["postAuthTokens"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/tokens/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    /**
     * Revoke an owned API token
     * @description Revoke an owned API token.
     */
    delete: operations["deleteAuthTokensId"];
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/providers": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List enabled external login providers
     * @description List enabled external login providers.
     */
    get: operations["getAuthProviders"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/identity-providers": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List identity provider configurations without secrets
     * @description List identity provider configurations without secrets. ADMIN cookie session with reauthentication within five minutes is required.
     */
    get: operations["getAdminIdentityProviders"];
    put?: never;
    /**
     * Create an identity provider
     * @description Create an identity provider. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminIdentityProviders"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/identity-providers/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    /**
     * Update an identity provider by revision
     * @description Current ADMIN cookie and fresh reauthentication, CAS and audit. Authority fields cannot change; update revokes provider sessions and pending flows. No implicit email linking or external role mapping. ADMIN cookie session with reauthentication within five minutes is required.
     */
    put: operations["putAdminIdentityProvidersId"];
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/providers/{id}/start": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Start a browser-bound SSO flow
     * @description Exact Origin. Link requires current cookie and fresh reauthentication; reauth requires current cookie. Returns IdP authorization URL, sets a short-lived HttpOnly browser-binding cookie. State, nonce and PKCE are persisted in PostgreSQL. Login does not require a session.
     */
    post: operations["postAuthProvidersIdStart"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/providers/{id}/callback": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Validate an OIDC authorization response
     * @description Code/state query; validates issuer, audience, signature, nonce, PKCE, verified email and fresh auth_time. Redirects only to configured origin root; the browser must complete the flow.
     */
    get: operations["getAuthProvidersIdCallback"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/providers/{id}/saml": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Validate a SAML POST response
     * @description Bounded SAMLResponse and RelayState form fields. Signed SP-initiated responses only; no Origin requirement for this cross-site endpoint. Redirects only to configured origin root, then browser binding is checked at completion.
     */
    post: operations["postAuthProvidersIdSaml"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/providers/{id}/metadata": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read SAML service provider metadata
     * @description Read SAML service provider metadata.
     */
    get: operations["getAuthProvidersIdMetadata"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/complete": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Atomically complete browser-bound external authentication
     * @description Exact Origin plus HttpOnly flow cookie. One-use pending identity, current provider revision, local role/disabled/epoch and original session for link/reauth checked in transaction. Returns a new local session and CSRF token. No automatic linking by email.
     */
    post: operations["postAuthComplete"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/providers/{id}/ldap": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Authenticate against a TLS LDAP directory
     * @description Exact Origin, rate limits, service search and nonempty user bind. Stable directory subject identifies the account; registration only if enabled, with USER role. Link/reauth require the original current cookie session.
     */
    post: operations["postAuthProvidersIdLdap"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/auth/identities": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List external identities linked to the current account
     * @description List external identities linked to the current account.
     */
    get: operations["getAuthIdentities"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/users": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Provision an account
     * @description Provision an account. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminUsers"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/settings": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List configuration revisions
     * @description List configuration revisions. ADMIN cookie session with reauthentication within five minutes is required.
     */
    get: operations["getAdminSettings"];
    put?: never;
    /**
     * Create a typed configuration draft
     * @description Create a typed configuration draft. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminSettings"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/settings/{revision}/validate": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Validate a configuration revision
     * @description Validate a configuration revision. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminSettingsRevisionValidate"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/settings/{revision}/apply": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Apply or roll back to a validated revision
     * @description Compare-and-set active revision, with an audit record in the same transaction. Select a previous validated revision to roll back. A changed networkPolicy or publicOrigin atomically closes the recovery gate, changes runtime generation, cancels existing requests and requires API/worker restart plus reconciliation/explicit reopening. Trusted proxy addresses are exact IPv4 /32 peers; requireHttps requires requireProxy and an HTTPS origin. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminSettingsRevisionApply"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/audit/verify": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Verify current audit segments and preserved checkpoint evidence
     * @description Verify current audit segments and preserved checkpoint evidence. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminAuditVerify"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/operations": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read private operational state and alerts
     * @description ADMIN cookie session and fresh reauthentication required. Never exposed by public status or owner notifications. ADMIN cookie session with reauthentication within five minutes is required.
     */
    get: operations["getAdminOperations"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/operations/maintenance": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Close the durable recovery gate
     * @description Close the durable recovery gate. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminOperationsMaintenance"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/operations/jobs": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Queue a backup, restore, reconciliation or integrity check
     * @description Restore requires a completed source backup and an enabled backup provider for the mandatory pre-restore copy. Closing the gate cancels and drains active handlers. Restore invalidates sessions and requires API/worker restart; no automatic reopen. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminOperationsJobs"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/operations/reconciliations/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read immutable recovery findings
     * @description Read immutable recovery findings. ADMIN cookie session with reauthentication within five minutes is required.
     */
    get: operations["getAdminOperationsReconciliationsId"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/operations/reopen": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Explicitly reopen after reviewed reconciliation
     * @description Requires matching current generation, current reconciliation ID/hash, age under 15 minutes, no unresolved critical incident and no pending operations. Explicitly acknowledge restored links/API tokens and quarantined files/orphans; this does not accept production residual risks. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminOperationsReopen"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/operations/capabilities/revoke": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Revoke all restored capabilities before reopening
     * @description Maintenance only. Revokes recipient links and API tokens, invalidates previous reconciliation. No owner notification. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminOperationsCapabilitiesRevoke"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/operations/checkpoints": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Create an exportable audit anchor
     * @description Recovery anchors require a closed gate and matching current full-chain evidence hash. Preserve damaged history; subsequent verification checks the preserved prefix hash and authenticates all subsequent events in both journals. Store the returned anchor independently. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminOperationsCheckpoints"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/operations/audit-evidence": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read the full audit history evidence hash
     * @description ADMIN-only hash of every historical audit row, including a damaged prefix. Preserve the evidence before creating an explicit recovery boundary. ADMIN cookie session with reauthentication within five minutes is required.
     */
    get: operations["getAdminOperationsAuditEvidence"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/operations/incidents/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    /**
     * Acknowledge or resolve an incident with an immutable report revision
     * @description Acknowledge or resolve an incident with an immutable report revision. ADMIN cookie session with reauthentication within five minutes is required.
     */
    patch: operations["patchAdminOperationsIncidentsId"];
    trace?: never;
  };
  "/api/v1/admin/operations/incidents/{id}/report": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read a private immutable security report
     * @description Query format=json|pdf, revision=0 for latest or a positive revision. No TTL or public verification; canonical JSON SHA-256 is returned in X-Content-SHA256. ADMIN cookie session with reauthentication within five minutes is required.
     */
    get: operations["getAdminOperationsIncidentsIdReport"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/operations/schedules": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    /**
     * Configure distributed operational schedules
     * @description Zero disables this optional schedule; otherwise minimum 60 seconds. Backup requires a provider. Mandatory existing cleanup schedules remain active when the data plane is open. ADMIN cookie session with reauthentication within five minutes is required.
     */
    put: operations["putAdminOperationsSchedules"];
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/exports": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List up to 100 latest exports for administrative review
     * @description Optional ownerId UUID query filter. Fresh ADMIN web reauthentication required; owner notifications are not generated. ADMIN cookie session with reauthentication within five minutes is required.
     */
    get: operations["getAdminExports"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/exports/{id}/credentials": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Issue an audited credential for another owner export
     * @description Only completed, ready, unexpired exports. Use the normal artifact claim/content flow. Every stream guard requires the ADMIN role and fresh reauthentication. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminExportsIdCredentials"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/exports/{id}/ttl": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    /**
     * Adjust an unexpired export artifact lifetime
     * @description Audited with reason. Lifetime is measured from now; deleted or expired artifacts cannot be resurrected. No owner notification. ADMIN cookie session with reauthentication within five minutes is required.
     */
    patch: operations["patchAdminExportsIdTtl"];
    trace?: never;
  };
  "/api/v1/transfers": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List own transfers by ID cursor
     * @description List own transfers by ID cursor.
     */
    get: operations["getTransfers"];
    put?: never;
    /**
     * Create a draft
     * @description Create a draft.
     */
    post: operations["postTransfers"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/{id}/webdav-credentials": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Create a WebDAV credential for one owner draft
     * @description Owner cookie session and reauthentication within five minutes required. Returns a password once, stored only as a digest in api_tokens with scope webdav:<transfer UUID>. Revoke via DELETE /auth/tokens/{id}. This credential grants no REST, recipient or cross-draft access; publication, freeze, expiry, revocation and account security changes deny access. See the x-webdav protocol profile for the non-JSON WebDAV methods.
     */
    post: operations["postTransfersIdWebdavCredentials"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/search": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Search own transfers using a private request body
     * @description Literal case-insensitive title search, state filter and stable createdAt/id cursor; no raw search text in the URL. A saved view is owner-scoped.
     */
    post: operations["postTransfersSearch"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read an owned transfer with links
     * @description Read an owned transfer with links.
     */
    get: operations["getTransfersId"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    /**
     * Edit permitted transfer metadata
     * @description Full metadata replacement; omitted title/message becomes empty. Payload names and paths remain immutable after publication.
     */
    patch: operations["patchTransfersId"];
    trace?: never;
  };
  "/api/v1/transfers/{id}/editor": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read owner transfer editor context
     * @description Uses the same owner isolation and transfers:read permission as transfer detail. Includes the saved message for full metadata replacement and the current publication TTL limit. This read neither mutates nor publishes the transfer; the original detail DTO remains unchanged.
     */
    get: operations["getTransfersIdEditor"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/{id}/publish": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Publish an immutable manifest and recipient links
     * @description Requires at least one ready file, no unfinished files and either a common link or recipients. Exact-input retry is idempotent. Changed revision/TTL/recipients/passwords conflicts. TTL starts at publication. Commit includes links, manifest, audit and outbox; it does not imply email delivery.
     */
    post: operations["postTransfersIdPublish"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/{id}/revoke": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Revoke recipient access and schedule payload deletion
     * @description Epoch changes interrupt active streams. Holds continue to prevent physical deletion.
     */
    post: operations["postTransfersIdRevoke"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/{id}/uploads": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Create a sticky resumable tus upload
     * @description tus 1.0 creation/termination, maximum file size 1 TiB. Upload-Metadata filename is a Base64 UTF-8 relative path. The selected provider and key remain sticky. Location identifies the resource.
     */
    post: operations["postTransfersIdUploads"];
    delete?: never;
    /**
     * Discover tus capabilities
     * @description Discover tus capabilities.
     */
    options: operations["optionsTransfersIdUploads"];
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/uploads/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    /**
     * Cancel a draft upload and queue deletion
     * @description Also accepts POST with X-HTTP-Method-Override: DELETE. No cancellation after publication.
     */
    delete: operations["deleteUploadsId"];
    /**
     * Discover tus resource capabilities
     * @description Discover tus resource capabilities.
     */
    options: operations["optionsUploadsId"];
    /**
     * Read the authoritative upload offset
     * @description Returns Upload-Offset and Upload-Length. Does not write bytes or finalize.
     */
    head: operations["headUploadsId"];
    /**
     * Append up to 4 MiB at the expected offset
     * @description Content-Type application/offset+octet-stream. Offset conflict returns 409. An empty PATCH is a no-op. Lost response: read HEAD before retrying. POST with X-HTTP-Method-Override: PATCH is equivalent.
     */
    patch: operations["patchUploadsId"];
    trace?: never;
  };
  "/api/v1/uploads/{id}/finalize": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Verify server checksum and file policy
     * @description The server reads authenticated chunks, computes SHA-256/CRC32, sniffs MIME and invokes an optional scanner. Client hashes or ready flags are never accepted.
     */
    post: operations["postUploadsIdFinalize"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/files/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    /**
     * Rename or move one draft file
     * @description Rename or move one draft file.
     */
    patch: operations["patchFilesId"];
    trace?: never;
  };
  "/api/v1/transfers/{id}/files": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    /**
     * Atomically rename or move a selection
     * @description Validates the complete resulting tree; a collision or invalid path rolls back every move. Published payload is immutable.
     */
    patch: operations["patchTransfersIdFiles"];
    trace?: never;
  };
  "/api/v1/transfers/{id}/folders": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List virtual folders
     * @description List virtual folders.
     */
    get: operations["getTransfersIdFolders"];
    put?: never;
    /**
     * Create a virtual draft folder
     * @description Create a virtual draft folder.
     */
    post: operations["postTransfersIdFolders"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/{id}/folders/{folder}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    /**
     * Delete a draft folder and cancel its uploads
     * @description All child paths are selected inside the parent transaction. Holds prevent physical cleanup.
     */
    delete: operations["deleteTransfersIdFoldersFolder"];
    options?: never;
    head?: never;
    /**
     * Recursively move a draft folder
     * @description Recursively move a draft folder.
     */
    patch: operations["patchTransfersIdFoldersFolder"];
    trace?: never;
  };
  "/api/v1/transfers/{id}/links/{link}/rotate": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Rotate a recipient capability and password
     * @description Immediately invalidates previous link epochs and recipient sessions. Omitted password removes the password requirement.
     */
    post: operations["postTransfersIdLinksLinkRotate"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/{id}/links/{link}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    /**
     * Revoke one recipient link
     * @description Revoke one recipient link.
     */
    delete: operations["deleteTransfersIdLinksLink"];
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/{id}/links/{link}/qr": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Download an owned capability QR code
     * @description Download an owned capability QR code.
     */
    get: operations["getTransfersIdLinksLinkQr"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/{id}/qr-bundle": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Create encrypted ZIP of recipient QR codes and email mapping
     * @description Sensitive temporary artifact, acquired through the artifact credential protocol. A concurrent link change rejects creation. Link rotation/revocation and transfer closure invalidate existing QR artifacts immediately.
     */
    post: operations["postTransfersIdQrBundle"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/{id}/journal": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read recipient-independent download totals
     * @description Contains no recipient identity, link ID, IP or user agent. One completed ZIP produces one aggregate row; totalBytes counts payload bytes.
     */
    get: operations["getTransfersIdJournal"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/recipient/session": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Exchange a capability and optional password for a session
     * @description Capability remains in URL fragment until the browser submits this request. Sets filexRecipient; exact Origin and per-capability throttling required. Does not consume files.
     */
    post: operations["postRecipientSession"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/recipient/files": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List session-bound opaque file references
     * @description References are stable within a recipient session and unusable in another session. No download acquisition on GET.
     */
    get: operations["getRecipientFiles"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/recipient/downloads": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Reserve an individual file or atomic ZIP
     * @description Choose exactly one of fileRef or all=true. All reserves every remaining ready file atomically; any held/unavailable file blocks the whole ZIP. Leases last five minutes and renew while streaming.
     */
    post: operations["postRecipientDownloads"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/recipient/downloads/{id}/content": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read reserved bytes with Range and fencing
     * @description Single byte range only. Complete successful server-observed range coverage commits logical deletion of the entire reservation. Revoke/expiry/hold guards run before and after each bounded 32 KiB write; already transmitted bytes cannot be retracted. HEAD never consumes.
     */
    get: operations["getRecipientDownloadsIdContent"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    /**
     * Inspect reserved content without consumption
     * @description Single byte range only. Complete successful server-observed range coverage commits logical deletion of the entire reservation. Revoke/expiry/hold guards run before and after each bounded 32 KiB write; already transmitted bytes cannot be retracted. HEAD never consumes.
     */
    head: operations["headRecipientDownloadsIdContent"];
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/transfers/{id}/hold": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Apply or release an indefinite retention hold
     * @description Empty fileId addresses the transfer. Transfer hold pauses ordinary TTL; absolute expiry never extends. USER receives no hold event. Unfreeze never resurrects revoked/expired rights. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminTransfersIdHold"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/transfers/{id}/forensic-zip": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Download a non-consuming forensic ZIP
     * @description Requires a transfer-wide hold, finalized retained files, current ADMIN session and fresh reauthentication. Loss of hold or privilege interrupts the stream. Writes an administrative audit record. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminTransfersIdForensicZip"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/files/{id}/forensic": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Download forensic plaintext or one raw ciphertext object
     * @description Raw ciphertext requires integrity_failed and a belonging objectKey. Plaintext authenticates every chunk. Neither consumes recipient rights; reason and actor are audited. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminFilesIdForensic"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/files/{id}/check": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Recheck payload integrity and conditionally repair state
     * @description Restores ready only when bytes match the original checksum and the parent is still active. Never reopens an expired/revoked transfer. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminFilesIdCheck"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/files/{id}/force-delete": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Force-delete retained file content
     * @description Irreversible. Requires transfer hold to be released; explicitly releases the selected file hold. Audited reason and reauthentication required. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminFilesIdForceDelete"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/storage": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List redacted provider metadata
     * @description List redacted provider metadata. ADMIN cookie session with reauthentication within five minutes is required.
     */
    get: operations["getAdminStorage"];
    put?: never;
    /**
     * Probe and register a storage provider
     * @description Writes, reads and deletes a generated probe. Configuration secrets are encrypted with the configuration DEK. Only one enabled artifact provider is allowed. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminStorage"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/storage/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    /**
     * Change provider priority, drain or enablement
     * @description Drain excludes new uploads; existing sticky uploads and reads continue. ADMIN cookie session with reauthentication within five minutes is required.
     */
    patch: operations["patchAdminStorageId"];
    trace?: never;
  };
  "/api/v1/admin/storage/migrations": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Preview or prepare a storage migration
     * @description dryRun=true returns 200 with files and bytes without mutation. Otherwise creates paused migration and drains the source. Target must be enabled, non-draining payload storage. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminStorageMigrations"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/storage/migrations/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    /**
     * Pause, resume or cancel a storage migration
     * @description Pause, resume or cancel a storage migration. ADMIN cookie session with reauthentication within five minutes is required.
     */
    patch: operations["patchAdminStorageMigrationsId"];
    trace?: never;
  };
  "/api/v1/admin/keys/rotate": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Rotate a class key and schedule re-encryption
     * @description New writes use the new key immediately. One running/paused rotation per class. Existing keys are retained; signing rotation keeps historical public keys permanently. ADMIN cookie session with reauthentication within five minutes is required.
     */
    post: operations["postAdminKeysRotate"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/keys/rotations/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    /**
     * Pause, resume or cancel background re-encryption
     * @description Does not undo the new active key or delete historical keys. A paused job prevents another rotation of that class. ADMIN cookie session with reauthentication within five minutes is required.
     */
    patch: operations["patchAdminKeysRotationsId"];
    trace?: never;
  };
  "/api/v1/admin/maintenance": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List migrations and rotations
     * @description List migrations and rotations. ADMIN cookie session with reauthentication within five minutes is required.
     */
    get: operations["getAdminMaintenance"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/signing-keys/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read a historical signing public key
     * @description publicKey is Base64 raw Ed25519 bytes. Verify trust out of band; the ZIP embeds SPKI PEM.
     */
    get: operations["getSigningKeysId"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/admin/smtp": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    /**
     * Save encrypted SMTP transport configuration
     * @description Only STARTTLS or SMTPS with certificate validation; authentication occurs after TLS. Saving is not proof of delivery. ADMIN cookie session with reauthentication within five minutes is required.
     */
    put: operations["putAdminSmtp"];
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/webhooks": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Create an outbound signed webhook
     * @description HTTPS port 443 only, no redirects or proxy. DNS resolved for every connection and all answers must be public IPv4. Empty secret generates a secret returned once. HMAC timestamp + body; consumers must deduplicate the signed eventId.
     */
    post: operations["postWebhooks"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/webhooks/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    /**
     * Disable an owned webhook
     * @description Disable an owned webhook.
     */
    delete: operations["deleteWebhooksId"];
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/notification-preferences": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read notification preferences
     * @description Read notification preferences.
     */
    get: operations["getNotificationPreferences"];
    /**
     * Set locale, timezone, digest and quiet hours
     * @description Set locale, timezone, digest and quiet hours.
     */
    put: operations["putNotificationPreferences"];
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/notifications": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Poll owner-safe notifications
     * @description Poll owner-safe notifications.
     */
    get: operations["getNotifications"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/notifications/{id}/read": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Mark an owned notification read
     * @description Mark an owned notification read.
     */
    post: operations["postNotificationsIdRead"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/events": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Subscribe to owner-safe WebSocket notifications
     * @description Cookie session and exact Origin only. after cursor resumes delivery; each event has schemaVersion=1. Session is rechecked every second. Poll /notifications as fallback. See AsyncAPI.
     */
    get: operations["getEvents"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/deliveries": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List own delivery states
     * @description List own delivery states.
     */
    get: operations["getDeliveries"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/deliveries/{id}/retry": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Resend a dead or previously delivered message
     * @description Resets bounded retry attempts; stable event ID allows receiver deduplication.
     */
    post: operations["postDeliveriesIdRetry"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/transfers/{id}/reports": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Create an immutable report revision
     * @description Snapshot uses REPEATABLE READ. USER snapshot excludes recipient correlation. ADMIN requires recent reauthentication and may include recipient association. PDF/CSV/JSON render asynchronously from the immutable encrypted snapshot.
     */
    post: operations["postTransfersIdReports"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/reports": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List owned report revisions
     * @description List owned report revisions.
     */
    get: operations["getReports"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/reports/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read a report snapshot, artifact IDs and history
     * @description Read a report snapshot, artifact IDs and history.
     */
    get: operations["getReportsId"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/reports/{id}/state": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Trash, restore or purge a report
     * @description Purge requires trashed state, deletes encrypted content asynchronously and preserves canonicalHash/history tombstones. Purged reports cannot be restored.
     */
    post: operations["postReportsIdState"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/reports/{id}/retry": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Retry rendering or regenerate expired report artifacts
     * @description Uses the existing immutable snapshot, never a new live-data snapshot. Available for failed rendering or expired/missing ready artifacts.
     */
    post: operations["postReportsIdRetry"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/verify/reports/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read the hash of an active USER report
     * @description Public allowlist response only. ADMIN, trashed and purged reports return the same neutral 404. Compare SHA-256 of the exact downloaded canonical JSON bytes.
     */
    get: operations["getVerifyReportsId"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/verify-report": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Compare a public USER report verification tuple
     * @description Exact report ID, revision and canonical hash comparison; active USER reports only. All absent, mismatching, ADMIN, trashed or purged cases return the same neutral 404.
     */
    post: operations["postVerifyReport"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/exports": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List own export jobs
     * @description List own export jobs.
     */
    get: operations["getExports"];
    put?: never;
    /**
     * Queue a consistent CSV or JSON export
     * @description Snapshot is taken at worker start; at most 50,000 rows / 16 MiB in a bounded REPEATABLE READ transaction. Limits fail explicitly without partial output. Global concurrency 2, per-owner 1; five-minute aging prevents USER starvation.
     */
    post: operations["postExports"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/exports/{id}/{action}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Cancel or retry an owned export
     * @description action=cancel or retry. Retained snapshot values are reused on retry; cancelled/expired staging is cleaned. Completed outputs expire after 24 hours.
     */
    post: operations["postExportsIdAction"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/artifacts/{id}/credentials": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Issue one-use session-bound artifact credentials
     * @description Cookie session only; credential expires after five minutes. ADMIN artifacts require current role and fresh reauthentication. Report trash/purge blocks issuing and streaming.
     */
    post: operations["postArtifactsIdCredentials"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/artifacts/{id}/claim": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Claim a credential once for resumable delivery
     * @description Sets path-scoped HttpOnly filexArtifact cookie. Subsequent Range requests also require the original filexSession. Coverage completion consumes the credential; owner can issue another while artifact is retained.
     */
    post: operations["postArtifactsIdClaim"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/artifacts/{id}/content": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read encrypted artifact through a fenced Range stream
     * @description HEAD never consumes; single byte Range supported. Every write rechecks the session, role, reauthentication for admin artifacts, artifact TTL and report state. Full coverage completes this credential.
     */
    get: operations["getArtifactsIdContent"];
    put?: never;
    post?: never;
    delete?: never;
    options?: never;
    /**
     * Read encrypted artifact through a fenced Range stream
     * @description HEAD never consumes; single byte Range supported. Every write rechecks the session, role, reauthentication for admin artifacts, artifact TTL and report state. Full coverage completes this credential.
     */
    head: operations["headArtifactsIdContent"];
    patch?: never;
    trace?: never;
  };
  "/api/v1/views": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List private saved views
     * @description List private saved views.
     */
    get: operations["getViews"];
    put?: never;
    /**
     * Save a private transfer view
     * @description Save a private transfer view.
     */
    post: operations["postViews"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/views/{id}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    post?: never;
    /**
     * Delete an owned saved view
     * @description Delete an owned saved view.
     */
    delete: operations["deleteViewsId"];
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/bulk": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * List own bulk jobs and progress
     * @description List own bulk jobs and progress.
     */
    get: operations["getBulk"];
    put?: never;
    /**
     * Freeze a server-side bulk selection
     * @description dryRun=true creates a paused selection snapshot. ADMIN scope requires reauthentication/reason; worker checks current role/account epoch. Each selected item is revalidated at execution and reports conflicts.
     */
    post: operations["postBulk"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/bulk/{id}/{action}": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    get?: never;
    put?: never;
    /**
     * Run or cancel a frozen bulk selection
     * @description action=run or cancel. A cancelled job stops before its next item; already completed item mutations remain.
     */
    post: operations["postBulkIdAction"];
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
  "/api/v1/setup": {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    /**
     * Read first-run state
     * @description Synchronous first-run setup with a host-provisioned expiring capability and durable completion latch. No asynchronous events are emitted.
     */
    get: {
      parameters: {
        query?: never;
        header?: never;
        path?: never;
        cookie?: never;
      };
      requestBody?: never;
      responses: {
        /** @description Current state; no secrets */
        200: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json": {
              /** @enum {unknown} */
              state: "disabled" | "pending" | "complete";
            };
          };
        };
        /** @description Invalid input */
        400: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
        /** @description Disabled, expired, token or Origin rejected */
        403: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
        /** @description Setup already completed */
        409: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
        /** @description Capacity reached */
        429: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
        /** @description Internal failure */
        500: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
        /** @description Database or identity provider unavailable */
        503: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
      };
    };
    put?: never;
    /**
     * Complete first-run setup once
     * @description Synchronous first-run setup with a host-provisioned expiring capability and durable completion latch. No asynchronous events are emitted.
     */
    post: {
      parameters: {
        query?: never;
        header: {
          Origin: string;
        };
        path?: never;
        cookie?: never;
      };
      requestBody: {
        content: {
          "application/json": components["schemas"]["ProductSetup"];
        };
      };
      responses: {
        /** @description Current state; no secrets */
        200: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json": {
              /** @enum {unknown} */
              state: "disabled" | "pending" | "complete";
            };
          };
        };
        /** @description Invalid input */
        400: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
        /** @description Disabled, expired, token or Origin rejected */
        403: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
        /** @description Setup already completed */
        409: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
        /** @description Capacity reached */
        429: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
        /** @description Internal failure */
        500: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
        /** @description Database or identity provider unavailable */
        503: {
          headers: {
            [name: string]: unknown;
          };
          content: {
            "application/json":
              | {
                  /** @enum {string} */
                  state:
                    | "disabled"
                    | "complete"
                    | "invalid_input"
                    | "identity_unavailable"
                    | "setup_failed"
                    | "unavailable";
                }
              | components["schemas"]["Error"];
          };
        };
      };
    };
    delete?: never;
    options?: never;
    head?: never;
    patch?: never;
    trace?: never;
  };
}
export type webhooks = Record<string, never>;
export interface components {
  schemas: {
    Error: {
      /** @enum {string} */
      code:
        | "unauthenticated"
        | "forbidden"
        | "not_found"
        | "conflict"
        | "invalid_input"
        | "rate_limited"
        | "internal_error"
        | "maintenance"
        | "restart_required";
      message: string;
    };
    Identifier: {
      /** Format: uuid */
      id: string;
    };
    Draft: {
      /** Format: uuid */
      id: string;
      revision: number;
    };
    Status: {
      /** @enum {string} */
      status: "operational" | "unavailable";
    };
    Principal: {
      /** Format: uuid */
      id: string;
      /** Format: email */
      email: string;
      /** @enum {string} */
      role: "user" | "admin";
      /** @enum {string} */
      locale: "ru" | "en";
      /** Format: uuid */
      sessionId?: string;
    };
    Session: {
      csrfToken: string;
      user: components["schemas"]["Principal"];
      /** Format: date-time */
      expiresAt: string;
    };
    NetworkPolicy: {
      trustedProxyCidrs?: string[];
      allowedClientCidrs?: string[];
      requireProxy?: boolean;
      requireHttps?: boolean;
    };
    Settings: {
      networkPolicy?: components["schemas"]["NetworkPolicy"];
      /** @constant */
      version: 1;
      publicOrigin?: string;
      sessionTtlSeconds?: number;
      transferMaxTtlSeconds: number;
      maxFiles: number;
      maxPathBytes: number;
      /** @enum {string} */
      fileTypeMode?: "disabled" | "allow" | "deny";
      fileTypes?: string[];
      bannerRu?: string;
      bannerEn?: string;
    };
    BrandPalette: {
      background: string;
      foreground: string;
      muted: string;
      surface: string;
      border: string;
      accent: string;
      onAccent: string;
      danger: string;
      dangerSurface: string;
    };
    Branding: {
      productName: string;
      taglineRu: string;
      taglineEn: string;
      footerText: string;
      /** @enum {string} */
      defaultTheme: "light" | "dark" | "system";
      /** @enum {string} */
      iconStyle: "outline" | "rounded" | "duotone";
      light: components["schemas"]["BrandPalette"];
      dark: components["schemas"]["BrandPalette"];
      /** @description Empty or a PNG/JPEG base64 data URL. Input and normalized PNG each at most 128 KiB; logos at most 1024x512 and favicon at most 256x256. Server decodes and re-encodes to PNG; URLs, SVG and metadata are not retained. */
      logoLight: string;
      /** @description Empty or a PNG/JPEG base64 data URL. Input and normalized PNG each at most 128 KiB; logos at most 1024x512 and favicon at most 256x256. Server decodes and re-encodes to PNG; URLs, SVG and metadata are not retained. */
      logoDark: string;
      /** @description Empty or a PNG/JPEG base64 data URL. Input and normalized PNG each at most 128 KiB; logos at most 1024x512 and favicon at most 256x256. Server decodes and re-encodes to PNG; URLs, SVG and metadata are not retained. */
      favicon: string;
    };
    BrandRevision: {
      revision: number;
      value: components["schemas"]["Branding"];
    };
    File: {
      /** Format: uuid */
      id: string;
      transferId: string;
      path: string;
      size: number;
      offset: number;
      /** @enum {string} */
      state:
        | "uploading"
        | "finalizing"
        | "ready"
        | "leased"
        | "cancelled"
        | "rejected"
        | "integrity_failed"
        | "storage_missing"
        | "deleting"
        | "deleted";
      sha256: string | null;
    };
    Link: {
      /** Format: uuid */
      id: string;
      email: string | null;
      url: string;
    };
    Transfer: {
      /** Format: uuid */
      id: string;
      revision: number;
      title: string;
      /** @enum {string} */
      state: "draft" | "active" | "completed" | "expired" | "revoked";
      manifestHash: string | null;
      files: components["schemas"]["File"][];
      links: components["schemas"]["Link"][];
    };
    TransferEditor: {
      /** Format: uuid */
      id: string;
      revision: number;
      title: string;
      /** @enum {string} */
      state: "draft" | "active" | "completed" | "expired" | "revoked";
      manifestHash: string | null;
      files: components["schemas"]["File"][];
      links: components["schemas"]["Link"][];
      message: string;
      /** @description Current installation limit for publication. The server rechecks this policy at publish time. */
      maxTtlSeconds: number;
    };
    Published: {
      /** Format: uuid */
      id: string;
      revision: number;
      /** Format: date-time */
      publishedAt: string;
      /** Format: date-time */
      expiresAt: string;
      /** Format: date-time */
      absoluteExpiresAt: string;
      manifestHash: string;
    };
    Filter: {
      query?: string;
      /** @enum {string} */
      state?: "" | "draft" | "active" | "completed" | "expired" | "revoked";
      after?: string;
      viewId?: string;
    };
    SearchResult: {
      items: {
        /** Format: uuid */
        id: string;
        title: string;
        /** @enum {string} */
        state: "draft" | "active" | "completed" | "expired" | "revoked";
        revision: number;
        /** Format: date-time */
        createdAt: string;
      }[];
      nextCursor: string;
    };
    JournalEntry: {
      /** Format: date-time */
      occurredAt: string;
      fileCount: number;
      totalBytes: number;
      status: string;
    };
    NotificationPreferences: {
      /** @enum {string} */
      locale: "ru" | "en";
      timezone: string;
      /** @enum {string} */
      digest: "immediate" | "hourly" | "daily" | "weekly" | "off";
      quietStart: number | null;
      quietEnd: number | null;
      browserEnabled: boolean;
    };
    OwnerEvent: {
      /** @constant */
      schemaVersion: 1;
      id: number;
      /** @enum {string} */
      kind:
        | "transfer.published"
        | "transfer.expiring"
        | "transfer.expired"
        | "transfer.revoked"
        | "download.completed";
      transferId: string | null;
      /** Format: date-time */
      createdAt: string;
      readAt: string | null;
    };
    WebhookEvent: {
      /** @constant */
      schemaVersion: 1;
      /** Format: uuid */
      eventId: string;
      /** @enum {string} */
      type:
        | "transfer.published"
        | "transfer.expiring"
        | "transfer.expired"
        | "transfer.revoked"
        | "download.completed";
      /** Format: uuid */
      transferId: string;
    };
    Delivery: {
      /** Format: uuid */
      id: string;
      eventType: string;
      /** @enum {string} */
      kind: "smtp" | "webhook";
      /** @enum {string} */
      state: "pending" | "sending" | "delivered" | "dead" | "cancelled";
      attempts: number;
      lastCode: string | null;
      /** Format: date-time */
      createdAt: string;
    };
    Report: {
      /** Format: uuid */
      id: string;
      /** Format: uuid */
      transferId: string;
      /** @enum {string} */
      kind: "user" | "admin";
      revision: number;
      canonicalHash: string;
      /** @enum {string} */
      state: "active" | "trashed" | "purged";
      /** @enum {string} */
      renderState: "pending" | "running" | "ready" | "failed";
      /** Format: date-time */
      createdAt: string;
    };
    ReportSnapshot: {
      /** @constant */
      schemaVersion: 1;
      /** Format: uuid */
      reportId: string;
      documentContentHash?: string;
      revision: number;
      /** @enum {string} */
      kind: "user" | "admin";
      /** Format: uuid */
      transferId: string;
      title: string;
      /** @enum {string} */
      state: "draft" | "active" | "completed" | "expired" | "revoked";
      publishedAt: string | null;
      manifestHash: string | null;
      /** Format: date-time */
      createdAt: string;
      /** @enum {string} */
      locale: "ru" | "en";
      timezone: string;
      manifest?: components["schemas"]["manifest-v1"];
      files: {
        path: string;
        size: number;
        sha256: string | null;
        /** @enum {string} */
        state:
          | "uploading"
          | "finalizing"
          | "ready"
          | "leased"
          | "cancelled"
          | "rejected"
          | "integrity_failed"
          | "storage_missing"
          | "deleting"
          | "deleted";
      }[];
      downloads: components["schemas"]["JournalEntry"][];
      administrativeEvents?: {
        /** Format: uuid */
        downloadId: string;
        recipientEmail: string | null;
        /** @enum {string} */
        kind: "file" | "zip";
        state: string;
        /** Format: date-time */
        createdAt: string;
        completedAt: string | null;
      }[];
    };
    ReportDetail: {
      /** Format: uuid */
      id: string;
      /** @enum {string} */
      state: "active" | "trashed" | "purged";
      /** @enum {string} */
      renderState: "pending" | "running" | "ready" | "failed";
      snapshot?: components["schemas"]["ReportSnapshot"];
      canonicalHash?: string;
      artifactHashes?: {
        [key: string]: string;
      };
      artifacts: {
        [key: string]: string;
      };
      history: {
        /** @enum {string} */
        action: "created" | "active" | "trashed" | "purged";
        /** Format: date-time */
        createdAt: string;
      }[];
    };
    Export: {
      /** Format: uuid */
      id: string;
      /** @enum {string} */
      kind: "user" | "admin";
      /** @enum {string} */
      format: "json" | "csv";
      /** @enum {string} */
      compression: "auto" | "none" | "gzip" | "zip";
      /** @enum {string} */
      state:
        | "queued"
        | "snapshotting"
        | "materialized"
        | "rendering"
        | "completed"
        | "failed"
        | "cancelled";
      rowCount: number;
      processed: number;
      snapshotAt: string | null;
      artifactId: string | null;
      errorCode: string | null;
      /** Format: date-time */
      createdAt: string;
    };
    Bulk: {
      /** Format: uuid */
      id: string;
      /** @enum {string} */
      action: "revoke" | "cancel_upload";
      /** @enum {string} */
      state: "paused" | "queued" | "running" | "completed" | "cancelled";
      total: number;
      done: number;
      conflicts: number;
    };
    SMTPConfiguration: {
      host: string;
      port: number;
      /** @enum {string} */
      tls: "starttls" | "smtps";
      username?: string;
      password?: string;
      /** Format: email */
      from: string;
    };
    S3Configuration: {
      endpoint?: string;
      region: string;
      bucket: string;
      accessKey: string;
      secretKey: string;
      sessionToken?: string;
    };
    StorageProvider: {
      /** Format: uuid */
      id: string;
      /** @enum {string} */
      kind: "local" | "s3";
      /** @enum {string} */
      purpose: "payload" | "artifacts" | "backups";
      priority: number;
      enabled: boolean;
      draining: boolean;
    };
    OperationsState: {
      /** @enum {string} */
      mode: "open" | "maintenance" | "restoring" | "reconcile" | "blocked";
      /** Format: uuid */
      generation: string;
      reason: string;
      restoreId: string | null;
      reconciliationId: string | null;
      /** Format: date-time */
      changedAt: string;
    };
    OperationsJob: {
      /** Format: uuid */
      id: string;
      /** @enum {string} */
      kind: "backup" | "restore" | "reconcile" | "audit" | "storage";
      /** @enum {string} */
      state: "queued" | "running" | "completed" | "failed";
      reason: string;
      sourceId: string | null;
      providerId: string | null;
      preBackupId: string | null;
      progress: number;
      errorCode: string | null;
      result: {
        [key: string]: unknown;
      };
      /** Format: date-time */
      createdAt: string;
      completedAt: string | null;
    };
    SecurityIncident: {
      /** Format: uuid */
      id: string;
      kind: string;
      subject: string;
      /** @enum {string} */
      severity: "warning" | "critical";
      /** @enum {string} */
      state: "open" | "acknowledged" | "resolved";
      /** Format: date-time */
      createdAt: string;
    };
    AuditCheckpoint: {
      /** Format: uuid */
      id: string;
      /** @enum {string} */
      kind: "regular" | "recovery";
      eventId: number;
      eventHash: string;
      evidenceHash: string;
      reason: string;
      /** Format: date-time */
      createdAt: string;
    };
    SignedCheckpoint: {
      checkpoint: components["schemas"]["AuditCheckpoint"];
      signature: string;
      operationsCheckpoint: components["schemas"]["AuditCheckpoint"] | null;
      operationsSignature: string | null;
    };
    OperationsSchedule: {
      /** @enum {string} */
      kind: "backup" | "audit" | "storage";
      intervalSeconds: number;
      /** Format: date-time */
      nextAt: string;
      providerId: string | null;
    };
    Reconciliation: {
      /** Format: uuid */
      id: string;
      /** Format: uuid */
      generation: string;
      /** Format: date-time */
      createdAt: string;
      files: number;
      artifacts: number;
      objects: number;
      links: number;
      tokens: number;
      findings: {
        kind: string;
        subject: string;
        /** @enum {string} */
        severity: "warning" | "critical";
      }[];
      passed: boolean;
      hash: string;
    };
    SecurityReport: {
      /** @constant */
      schemaVersion: 1;
      /** Format: uuid */
      id: string;
      /** Format: uuid */
      incidentId: string;
      revision: number;
      kind: string;
      subject: string;
      /** @enum {string} */
      severity: "warning" | "critical";
      /** @enum {string} */
      state: "open" | "acknowledged" | "resolved";
      reason: string;
      evidence: {
        [key: string]: unknown;
      };
      /** Format: date-time */
      createdAt: string;
    };
    LicenseClaims: {
      /** @constant */
      schemaVersion: 1;
      /** @constant */
      product: "filex";
      /** @constant */
      issuer: "filex-vendor";
      /** Format: uuid */
      licenseId: string;
      /** Format: uuid */
      entitlementId: string;
      revision: number;
      customerId: string;
      /** Format: uuid */
      installationId: string;
      /** @enum {unknown} */
      kind: "subscription" | "perpetual";
      /** Format: date-time */
      issuedAt: string;
      /** Format: date-time */
      notBefore: string;
      expiresAt: string | null;
      /** Format: date-time */
      updatesUntil: string;
      /** Format: date-time */
      supportUntil: string;
      limits: {
        maxEnabledUsers: number | null;
        maxTransferBytes: number | null;
      };
      features: ("webdav" | "federated_auth" | "white_label")[];
    } & unknown;
    LicenseStatus: {
      /** Format: uuid */
      installationId: string;
      epoch: number;
      /** @enum {string} */
      state:
        | "missing"
        | "active"
        | "expired"
        | "invalid"
        | "not_yet_valid"
        | "clock_suspect"
        | "update_not_entitled";
      claims: components["schemas"]["LicenseClaims"] | null;
      pending: components["schemas"]["LicenseClaims"] | null;
      enabledUsers: number;
      overLimit: boolean;
    };
    IdentityConfig: {
      trustEmail: boolean;
      hostedDomain: string;
      name: string;
      /** @enum {string} */
      protocol: "oidc" | "saml" | "ldap";
      enabled: boolean;
      allowRegistration: boolean;
      allowedEmailDomains: string[];
      privateCidrs: string[];
      caPem: string;
      issuer: string;
      clientId: string;
      extraOrigins: string[];
      metadataXml: string;
      emailAttribute: string;
      ldapUrl: string;
      baseDn: string;
      bindDn: string;
      loginAttribute: string;
      subjectAttribute: string;
    };
    IdentityProvider: {
      /** Format: uuid */
      id: string;
      revision: number;
      config: components["schemas"]["IdentityConfig"];
      hasSecret: boolean;
    };
    IdentityPublic: {
      /** Format: uuid */
      id: string;
      name: string;
      /** @enum {string} */
      protocol: "oidc" | "saml" | "ldap";
      allowRegistration: boolean;
    };
    IdentityInput: {
      expectedRevision: number;
      config: components["schemas"]["IdentityConfig"];
      /** @description Omitted preserves the encrypted secret; empty explicitly clears it. */
      secret?: string;
    };
    WebDAVCredential: {
      /** Format: uuid */
      id: string;
      /** Format: uri */
      url: string;
      /** @constant */
      username: "filex";
      password: string;
      /** Format: date-time */
      expiresAt: string;
    };
    ProductSetup: {
      /** Format: email */
      email: string;
      password: string;
    };
    SetupStatus: {
      /** @enum {unknown} */
      state: "disabled" | "pending" | "complete";
    };
    "manifest-v1": {
      /** @constant */
      version: 1;
      files: {
        /** Format: uuid */
        id: string;
        path: string;
        size: number;
        sha256: string;
      }[];
    };
  };
  responses: never;
  parameters: never;
  requestBodies: never;
  headers: never;
  pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
  getStatus: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Status"];
        };
      };
      /** @description Unavailable */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Status"];
        };
      };
    };
  };
  getBranding: {
    parameters: {
      query?: never;
      header?: {
        /** @description ETag returned by an earlier branding response. */
        "If-None-Match"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          ETag?: string;
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["BrandRevision"];
        };
      };
      /** @description Branding is unchanged; no response body. */
      304: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAdminBranding: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["BrandRevision"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  putAdminBranding: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          expectedRevision: number;
          value: components["schemas"]["Branding"];
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["BrandRevision"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAdminLicense: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["LicenseStatus"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  putAdminLicense: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          envelope: string;
          expectedEpoch: number;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["LicenseStatus"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminLicensePreview: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          envelope: string;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["LicenseClaims"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getBanner: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            ru: string;
            en: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAuthLogin: {
    parameters: {
      query?: never;
      header: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          email: string;
          /** @description Account password: 12–1024 UTF-8 bytes. Link secrets also accept a numeric PIN of 6–12 digits. */
          password: string;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Session"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAuthMe: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Principal"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAuthLogout: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAuthReauth: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @description Account password: 12–1024 UTF-8 bytes. Link secrets also accept a numeric PIN of 6–12 digits. */
          password: string;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  patchAuthPassword: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @description Account password: 12–1024 UTF-8 bytes. Link secrets also accept a numeric PIN of 6–12 digits. */
          currentPassword: string;
          /** @description Account password: 12–1024 UTF-8 bytes. Link secrets also accept a numeric PIN of 6–12 digits. */
          newPassword: string;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAuthSessions: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            /** Format: date-time */
            createdAt: string;
            /** Format: date-time */
            expiresAt: string;
            revokedAt: string | null;
          }[];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  deleteAuthSessionsId: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAuthHistory: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: date-time */
            occurredAt: string;
            success: boolean;
          }[];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAuthTokens: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            scopes: string[];
            /** Format: date-time */
            expiresAt: string;
            revokedAt: string | null;
          }[];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAuthTokens: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          scopes: string[];
          ttlSeconds: number;
        };
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            token: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  deleteAuthTokensId: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAuthProviders: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["IdentityPublic"][];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAdminIdentityProviders: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["IdentityProvider"][];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminIdentityProviders: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["IdentityInput"];
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["IdentityProvider"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  putAdminIdentityProvidersId: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["IdentityInput"];
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["IdentityProvider"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAuthProvidersIdStart: {
    parameters: {
      query?: never;
      header: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          purpose: "login" | "link" | "reauth";
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            url: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAuthProvidersIdCallback: {
    parameters: {
      query: {
        code: string;
        state: string;
      };
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      303: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAuthProvidersIdSaml: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/x-www-form-urlencoded": {
          SAMLResponse: string;
          RelayState: string;
        };
      };
    };
    responses: {
      /** @description Success */
      303: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAuthProvidersIdMetadata: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/samlmetadata+xml": string;
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAuthComplete: {
    parameters: {
      query?: never;
      header: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          state: string;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Session"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAuthProvidersIdLdap: {
    parameters: {
      query?: never;
      header: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          username: string;
          password: string;
          /** @enum {string} */
          purpose: "login" | "link" | "reauth";
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Session"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Authentication rejected without private details */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAuthIdentities: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            providerId: string;
            name: string;
            /** @enum {string} */
            protocol: "oidc" | "saml" | "ldap";
            enabled: boolean;
          }[];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminUsers: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          email: string;
          /** @description Account password: 12–1024 UTF-8 bytes. Link secrets also accept a numeric PIN of 6–12 digits. */
          password: string;
          /** @enum {string} */
          role: "user" | "admin";
        };
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Identifier"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAdminSettings: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            revision: number;
            active: boolean;
            validated: boolean;
            value: components["schemas"]["Settings"];
          }[];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminSettings: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["Settings"];
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            revision: number;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminSettingsRevisionValidate: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        revision: number;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminSettingsRevisionApply: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        revision: number;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          expectedRevision: number;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminAuditVerify: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** @constant */
            status: "verified";
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAdminOperations: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            state: components["schemas"]["OperationsState"];
            jobs: components["schemas"]["OperationsJob"][];
            incidents: components["schemas"]["SecurityIncident"][];
            checkpoints: components["schemas"]["SignedCheckpoint"][];
            schedules: components["schemas"]["OperationsSchedule"][];
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminOperationsMaintenance: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminOperationsJobs: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          kind: "backup" | "restore" | "reconcile" | "audit" | "storage";
          /** Format: uuid */
          sourceId?: string;
          /** Format: uuid */
          providerId?: string;
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
        };
      };
    };
    responses: {
      /** @description Success */
      202: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Identifier"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAdminOperationsReconciliationsId: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Reconciliation"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminOperationsReopen: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** Format: uuid */
          reconciliationId: string;
          hash: string;
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
          acknowledgeCapabilities: boolean;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminOperationsCapabilitiesRevoke: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminOperationsCheckpoints: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          kind: "regular" | "recovery";
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
          evidenceHash?: string;
        };
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["SignedCheckpoint"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAdminOperationsAuditEvidence: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            evidenceHash: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  patchAdminOperationsIncidentsId: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          state: "acknowledged" | "resolved";
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAdminOperationsIncidentsIdReport: {
    parameters: {
      query?: {
        /** @description Defaults to json. */
        format?: "json" | "pdf";
        /** @description Zero or omitted selects latest revision. */
        revision?: number;
      };
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          /** @description SHA-256 of the immutable source JSON, including for a PDF response. */
          "X-Content-SHA256"?: string;
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["SecurityReport"];
          "application/pdf": string;
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  putAdminOperationsSchedules: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          kind: "backup" | "audit" | "storage";
          intervalSeconds: 0 | number;
          /** Format: uuid */
          providerId?: string;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAdminExports: {
    parameters: {
      query?: {
        /** @description Optional exact owner UUID filter. */
        ownerId?: string;
      };
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            /** Format: uuid */
            ownerId: string;
            /** @enum {string} */
            kind: "user" | "admin";
            /** @enum {string} */
            format: "json" | "csv";
            /** @enum {string} */
            state:
              | "queued"
              | "snapshotting"
              | "materialized"
              | "rendering"
              | "completed"
              | "failed"
              | "cancelled";
            artifactId: string | null;
            expiresAt: string | null;
            /** Format: date-time */
            createdAt: string;
          }[];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminExportsIdCredentials: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
        };
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            artifactId: string;
            token: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  patchAdminExportsIdTtl: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
          ttlSeconds: number;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getTransfers: {
    parameters: {
      query?: {
        /** @description nextCursor from the previous result. */
        cursor?: string;
      };
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            items: {
              /** Format: uuid */
              id: string;
              /** @enum {string} */
              state: "draft" | "active" | "completed" | "expired" | "revoked";
              revision: number;
              title: string;
              manifestHash: string | null;
            }[];
            nextCursor: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postTransfers: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Draft"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postTransfersIdWebdavCredentials: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          ttlSeconds: number;
        };
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["WebDAVCredential"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postTransfersSearch: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["Filter"];
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["SearchResult"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getTransfersId: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Transfer"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  patchTransfersId: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          title?: string;
          message?: string;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getTransfersIdEditor: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["TransferEditor"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postTransfersIdPublish: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          expectedRevision: number;
          ttlSeconds: number;
          recipients?: {
            email: string;
            /** @description Account password: 12–1024 UTF-8 bytes. Link secrets also accept a numeric PIN of 6–12 digits. */
            password?: string;
          }[];
          commonLink?: boolean;
          /** @description Account password: 12–1024 UTF-8 bytes. Link secrets also accept a numeric PIN of 6–12 digits. */
          commonPassword?: string;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            transfer: components["schemas"]["Published"];
            links: components["schemas"]["Link"][];
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postTransfersIdRevoke: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postTransfersIdUploads: {
    parameters: {
      query?: never;
      header: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
        "Tus-Resumable": "1.0.0";
        "Upload-Length": number;
        /** @example filename YS50eHQ= */
        "Upload-Metadata": string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      201: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          Location?: string;
          "Upload-Offset"?: number;
          "Upload-Length"?: number;
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Draft"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing/unsupported tus version; Tus-Version: 1.0.0. */
      412: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  optionsTransfersIdUploads: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          "Upload-Offset"?: number;
          "Upload-Length"?: number;
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  deleteUploadsId: {
    parameters: {
      query?: never;
      header: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
        "Tus-Resumable": "1.0.0";
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          "Upload-Offset"?: number;
          "Upload-Length"?: number;
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing/unsupported tus version; Tus-Version: 1.0.0. */
      412: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  optionsUploadsId: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          "Upload-Offset"?: number;
          "Upload-Length"?: number;
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  headUploadsId: {
    parameters: {
      query?: never;
      header: {
        "Tus-Resumable": "1.0.0";
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          "Upload-Offset"?: number;
          "Upload-Length"?: number;
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing/unsupported tus version; Tus-Version: 1.0.0. */
      412: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  patchUploadsId: {
    parameters: {
      query?: never;
      header: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
        "Tus-Resumable": "1.0.0";
        "Upload-Offset": number;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/offset+octet-stream": string;
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          "Upload-Offset"?: number;
          "Upload-Length"?: number;
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing/unsupported tus version; Tus-Version: 1.0.0. */
      412: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Chunk larger than 4 MiB */
      413: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Unsupported PATCH media type */
      415: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          "Tus-Resumable"?: "1.0.0";
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postUploadsIdFinalize: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  patchFilesId: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          path: string;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  patchTransfersIdFiles: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          moves: {
            /** Format: uuid */
            id: string;
            path: string;
          }[];
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getTransfersIdFolders: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            path: string;
          }[];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postTransfersIdFolders: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          path: string;
        };
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            path: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  deleteTransfersIdFoldersFolder: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
        /** @description Resource identifier or allowed action. */
        folder: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  patchTransfersIdFoldersFolder: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
        /** @description Resource identifier or allowed action. */
        folder: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          path: string;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postTransfersIdLinksLinkRotate: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
        /** @description Resource identifier or allowed action. */
        link: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @description Account password: 12–1024 UTF-8 bytes. Link secrets also accept a numeric PIN of 6–12 digits. */
          password?: string;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            url: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  deleteTransfersIdLinksLink: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
        /** @description Resource identifier or allowed action. */
        link: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getTransfersIdLinksLinkQr: {
    parameters: {
      query?: {
        format?: "png" | "svg";
      };
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
        /** @description Resource identifier or allowed action. */
        link: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/octet-stream": string;
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postTransfersIdQrBundle: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            artifactId: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getTransfersIdJournal: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["JournalEntry"][];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postRecipientSession: {
    parameters: {
      query?: never;
      header: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          token: string;
          /** @description Account password: 12–1024 UTF-8 bytes. Link secrets also accept a numeric PIN of 6–12 digits. */
          password?: string;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** @constant */
            status: "ready";
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getRecipientFiles: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            ref: string;
            path: string;
            size: number;
            /** @enum {string} */
            state:
              | "uploading"
              | "finalizing"
              | "ready"
              | "leased"
              | "cancelled"
              | "rejected"
              | "integrity_failed"
              | "storage_missing"
              | "deleting"
              | "deleted";
          }[];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postRecipientDownloads: {
    parameters: {
      query?: never;
      header: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          fileRef?: string;
          all?: boolean;
        };
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            url: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getRecipientDownloadsIdContent: {
    parameters: {
      query?: never;
      header?: {
        /** @description One range only; suffix ranges accepted. */
        Range?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          "Accept-Ranges"?: "bytes";
          "Content-Length"?: number;
          "Content-Disposition"?: string;
          [name: string]: unknown;
        };
        content: {
          "application/octet-stream": string;
        };
      };
      /** @description Partial content */
      206: {
        headers: {
          "Content-Range"?: string;
          "Content-Length"?: number;
          "Accept-Ranges"?: "bytes";
          [name: string]: unknown;
        };
        content: {
          "application/octet-stream": string;
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Unsatisfiable range; Content-Range contains total length. */
      416: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  headRecipientDownloadsIdContent: {
    parameters: {
      query?: never;
      header?: {
        /** @description One range only; suffix ranges accepted. */
        Range?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          "Accept-Ranges"?: "bytes";
          "Content-Length"?: number;
          "Content-Disposition"?: string;
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Partial content */
      206: {
        headers: {
          "Content-Range"?: string;
          "Content-Length"?: number;
          "Accept-Ranges"?: "bytes";
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Unsatisfiable range; Content-Range contains total length. */
      416: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminTransfersIdHold: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          frozen: boolean;
          fileId?: string;
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminTransfersIdForensicZip: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
        };
        "application/x-www-form-urlencoded": {
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
          _csrf: string;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/octet-stream": string;
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminFilesIdForensic: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
          rawCiphertext?: boolean;
          objectKey?: string;
        };
        "application/x-www-form-urlencoded": {
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
          _csrf: string;
          /** @enum {string} */
          rawCiphertext?: "true" | "false";
          objectKey?: string;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/octet-stream": string;
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminFilesIdCheck: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            status: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminFilesIdForceDelete: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason: string;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAdminStorage: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["StorageProvider"][];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminStorage: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          kind: "local" | "s3";
          /** @enum {string} */
          purpose?: "payload" | "artifacts";
          root?: string;
          s3?: components["schemas"]["S3Configuration"];
          priority?: number;
        };
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Identifier"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  patchAdminStorageId: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          priority: number;
          draining: boolean;
          enabled: boolean;
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminStorageMigrations: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** Format: uuid */
          sourceId: string;
          /** Format: uuid */
          targetId: string;
          dryRun: boolean;
        };
      };
    };
    responses: {
      /** @description Read-only dry-run totals */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            files: number;
            bytes: number;
          };
        };
      };
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            /** @constant */
            state: "paused";
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  patchAdminStorageMigrationsId: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          state: "paused" | "running" | "cancelled";
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postAdminKeysRotate: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          class:
            "payload" | "artifacts" | "configuration" | "signing" | "backups";
        };
      };
    };
    responses: {
      /** @description Success */
      202: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            /** Format: uuid */
            keyId: string;
            /** @enum {string} */
            state: "running" | "completed";
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  patchAdminKeysRotationsId: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          state: "paused" | "running" | "cancelled";
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getAdminMaintenance: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            kind: string;
            state: string;
            progress: number;
          }[];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getSigningKeysId: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            publicKey: string;
            /** @constant */
            algorithm: "Ed25519";
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  putAdminSmtp: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["SMTPConfiguration"];
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            /** @constant */
            status: "configured";
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postWebhooks: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          url: string;
          secret?: string;
        };
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            secret: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  deleteWebhooksId: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getNotificationPreferences: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["NotificationPreferences"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  putNotificationPreferences: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": components["schemas"]["NotificationPreferences"];
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getNotifications: {
    parameters: {
      query?: {
        /** @description Last received monotonic notification ID. */
        after?: number;
      };
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["OwnerEvent"][];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postNotificationsIdRead: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getEvents: {
    parameters: {
      query?: {
        /** @description Last received monotonic notification ID. */
        after?: number;
      };
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      101: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getDeliveries: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Delivery"][];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postDeliveriesIdRetry: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postTransfersIdReports: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          kind: "user" | "admin";
          /** @enum {string} */
          locale: "ru" | "en";
          timezone: string;
        };
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            canonicalHash: string;
            /** @constant */
            renderState: "pending";
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getReports: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Report"][];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getReportsId: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["ReportDetail"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postReportsIdState: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          state: "active" | "trashed" | "purged";
        };
      };
    };
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postReportsIdRetry: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getVerifyReportsId: {
    parameters: {
      query?: never;
      header?: never;
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            canonicalHash: string;
            /** @enum {string} */
            state: "active" | "trashed" | "purged";
            revision: number;
            /** Format: date-time */
            createdAt: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postVerifyReport: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** Format: uuid */
          reportId: string;
          revision: number;
          canonicalHash: string;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** @constant */
            verified: true;
            /** Format: uuid */
            reportId: string;
            revision: number;
            canonicalHash: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getExports: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Export"][];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postExports: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          /** @enum {string} */
          kind: "user" | "admin";
          /** @enum {string} */
          format: "csv" | "json";
          /** @enum {string} */
          compression: "auto" | "none" | "gzip" | "zip";
        };
      };
    };
    responses: {
      /** @description Success */
      202: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            /** @constant */
            state: "queued";
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postExportsIdAction: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
        /** @description Resource identifier or allowed action. */
        action: "cancel" | "retry";
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postArtifactsIdCredentials: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            token: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postArtifactsIdClaim: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          token: string;
        };
      };
    };
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            url: string;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getArtifactsIdContent: {
    parameters: {
      query?: never;
      header?: {
        /** @description One range only; suffix ranges accepted. */
        Range?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          "Accept-Ranges"?: "bytes";
          "Content-Length"?: number;
          "Content-Disposition"?: string;
          [name: string]: unknown;
        };
        content: {
          "application/octet-stream": string;
        };
      };
      /** @description Partial content */
      206: {
        headers: {
          "Content-Range"?: string;
          "Content-Length"?: number;
          "Accept-Ranges"?: "bytes";
          [name: string]: unknown;
        };
        content: {
          "application/octet-stream": string;
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Unsatisfiable range; Content-Range contains total length. */
      416: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  headArtifactsIdContent: {
    parameters: {
      query?: never;
      header?: {
        /** @description One range only; suffix ranges accepted. */
        Range?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          "Accept-Ranges"?: "bytes";
          "Content-Length"?: number;
          "Content-Disposition"?: string;
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Partial content */
      206: {
        headers: {
          "Content-Range"?: string;
          "Content-Length"?: number;
          "Accept-Ranges"?: "bytes";
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Unsatisfiable range; Content-Range contains total length. */
      416: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getViews: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            name: string;
            filter: components["schemas"]["Filter"];
          }[];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postViews: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          name: string;
          filter: components["schemas"]["Filter"];
        };
      };
    };
    responses: {
      /** @description Success */
      201: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Identifier"];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  deleteViewsId: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  getBulk: {
    parameters: {
      query?: never;
      header?: never;
      path?: never;
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      200: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Bulk"][];
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postBulk: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path?: never;
      cookie?: never;
    };
    requestBody: {
      content: {
        "application/json": {
          ids: string[];
          /** @enum {string} */
          action: "revoke" | "cancel_upload";
          adminScope?: boolean;
          /** @description 5 to 2000 Unicode characters; at least 5 characters must remain after trimming surrounding whitespace. */
          reason?: string;
          dryRun?: boolean;
        };
      };
    };
    responses: {
      /** @description Success */
      202: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": {
            /** Format: uuid */
            id: string;
            /** @enum {string} */
            state: "paused" | "queued";
            selected: number;
          };
        };
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
  postBulkIdAction: {
    parameters: {
      query?: never;
      header?: {
        /** @description Must equal configured publicOrigin for all cookie and public authentication requests; not required for bearer authentication. */
        Origin?: string;
        /** @description Required with cookie authentication; obtained at login. Not required for bearer requests. */
        "X-CSRF-Token"?: string;
      };
      path: {
        /** @description Resource identifier or allowed action. */
        id: string;
        /** @description Resource identifier or allowed action. */
        action: "run" | "cancel";
      };
      cookie?: never;
    };
    requestBody?: never;
    responses: {
      /** @description Success */
      204: {
        headers: {
          [name: string]: unknown;
        };
        content?: never;
      };
      /** @description Invalid input */
      400: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Missing, expired or revoked authentication */
      401: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Origin, CSRF, role, reauthentication, ownership or scope rejected */
      403: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Resource unavailable */
      404: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description State, offset, revision, lease or key changed */
      409: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Rate or concurrency limit; retry later */
      429: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Fail-closed internal failure without private details */
      500: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
      /** @description Maintenance or runtime restart required */
      503: {
        headers: {
          [name: string]: unknown;
        };
        content: {
          "application/json": components["schemas"]["Error"];
        };
      };
    };
  };
}
