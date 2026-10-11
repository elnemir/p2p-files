import unittest
from acceptance import decision_gaps,gate_gaps,DECISIONS

class ReleaseAcceptanceTest(unittest.TestCase):
    def test_missing_duplicate_and_pending_decisions_block_release(self):
        self.assertTrue(decision_gaps({}))
        rows=[{'id':name,'status':'pending'} for name in DECISIONS]
        self.assertEqual(len(decision_gaps({'version':1,'decisions':rows})),len(DECISIONS))
        rows[-1]=rows[0]
        self.assertTrue(decision_gaps({'version':1,'decisions':rows}))
    def test_acceptance_without_reviewer_or_evidence_is_not_approval(self):
        rows=[{'id':name,'status':'accepted','reviewer':'','evidence':''} for name in DECISIONS]
        self.assertEqual(len(decision_gaps({'version':1,'decisions':rows})),len(DECISIONS))
        for row in rows:row.update(reviewer='fixture reviewer',evidence='fixture evidence reference')
        self.assertEqual(decision_gaps({'version':1,'decisions':rows}),[])

    def test_stale_failed_partial_or_compact_evidence_blocks_release(self):
        result={'gate':'performance','passed':True,'exitCode':0,'sourceUnchanged':True,'sourceSha256':'candidate','finishedAtUnix':100000,'profile':{'FILEX_PERF_MIB':'2050','FILEX_PERF_ROWS':'20000'}}
        self.assertEqual(gate_gaps('performance',result,'candidate',100001),[])
        for change in [{'sourceSha256':'old'},{'exitCode':1},{'sourceUnchanged':False},{'finishedAtUnix':1},{'finishedAtUnix':100002},{'profile':{}}]:
            self.assertTrue(gate_gaps('performance',dict(result,**change),'candidate',100001))
        restricted=dict(result,gate='browser',profile={'FILEX_E2E_PROJECTS':'chromium'})
        self.assertTrue(gate_gaps('browser',restricted,'candidate',100001))

if __name__=='__main__':unittest.main()
