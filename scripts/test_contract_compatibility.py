import unittest
from contract_compatibility import canonical,differences

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

if __name__=='__main__':unittest.main()
