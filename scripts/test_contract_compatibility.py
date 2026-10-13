import unittest
from contract_compatibility import canonical,differences,new_schema_findings

class ContractCompatibilityTest(unittest.TestCase):
    def test_removal_and_type_changes_fail(self):
        old={'paths':{'/v1/item':{'get':{'type':'string'}}}}
        self.assertTrue(differences(old,{'paths':{}}))
        self.assertTrue(differences(old,{'paths':{'/v1/item':{'get':{'type':'number'}}}}))
    def test_required_enum_and_output_changes_fail(self):
        for before,after in [({'required':['x']},{'required':[]}),({'enum':['a','b']},{'enum':['a']}),({'properties':{'x':{}}},{'properties':{'x':{},'y':{}}})]:
            self.assertTrue(differences(before,after))
    def test_documentation_and_new_paths_are_compatible(self):
        old={'paths':{'/v1/item':{'get':{'description':'Old'}}}}
        new={'paths':{'/v1/item':{'get':{'description':'Clarified'}},'/v1/new':{}}}
        self.assertEqual(differences(canonical(old),canonical(new)),[])

    def test_schema_property_names_and_literal_values_are_contract_data(self):
        for before,after in [({'properties':{'description':{'type':'string'}}},{'properties':{}}),({'enum':[{'description':'a'}]},{'enum':[{'description':'b'}]}),({'default':{'examples':'a'}},{'default':{'examples':'b'}})]:
            self.assertTrue(differences(canonical(before),canonical(after)))

    def test_additive_auth_and_protocol_do_not_relax_existing_contracts(self):
        old={'components':{'securitySchemes':{'session':{'type':'apiKey'}}}}
        new={'components':{'securitySchemes':{'session':{'type':'apiKey'},'webdav':{'type':'http','scheme':'basic'}}},'x-webdav':{'profile':'v1'}}
        self.assertEqual(differences(old,new),[])
        changed={'components':{'securitySchemes':{'session':{'type':'http'},'webdav':{'type':'http','scheme':'basic'}}}}
        self.assertTrue(differences(old,changed))
        self.assertTrue(differences({'x-webdav':{'profile':'v1'}},{'x-webdav':{'profile':'v2'}}))
        self.assertTrue(differences(old,{'components':{'securitySchemes':{}}}))

    def test_new_schema_requires_distinct_identity_and_valid_dialect(self):
        existing={'old.json':{'$id':'https://filex.invalid/schemas/old-v1.json'}}
        valid={'$schema':'https://json-schema.org/draft/2020-12/schema','$id':'https://filex.invalid/schemas/new-v1.json','type':'object'}
        self.assertEqual(new_schema_findings(existing,{'new.json':valid}),[])
        for value in [None,{},[],{**valid,'$schema':'unknown'},{**valid,'$id':existing['old.json']['$id']},{**valid,'$id':'relative-id'}]:
            self.assertTrue(new_schema_findings(existing,{'new.json':value}))
        self.assertTrue(new_schema_findings(existing,{'first.json':valid,'duplicate.json':valid}))

if __name__=='__main__':unittest.main()
