"""First-run contracts; no asynchronous setup work or new broker events."""
def extend(outputs):
    def obj(properties):
        return {'type':'object','additionalProperties':False,'required':list(properties),'properties':properties}
    text={'type':'string','minLength':1,'maxLength':2048}
    product=obj({'email':{'type':'string','format':'email','maxLength':254},'password':{'type':'string','minLength':12,'maxLength':1024}})
    issuer=obj({k:text for k in ['issuer','clientId','clientSecret','requiredAcr','subject']})
    status=obj({'state':{'enum':['disabled','pending','complete']}})
    for file,path,name,body in [('contracts/openapi.json','/api/v1/setup','ProductSetup',product),('contracts/issuer-openapi.json','/issuer/api/setup','IssuerSetup',issuer)]:
        api=outputs[file]
        api['components']['schemas'].update({name:body,'SetupStatus':status})
        api['components']['securitySchemes']['setupToken']={'type':'apiKey','in':'header','name':'X-Setup-Token','description':'Host-provisioned random token; expiring and disabled permanently after completion.'}
        api['paths'][path]={}
        for method in ['get','post']:
            op={'x-implementation':'implemented','description':'Synchronous first-run setup with a host-provisioned expiring capability and durable completion latch. No asynchronous events are emitted.','summary':'Read first-run state' if method=='get' else 'Complete first-run setup once','security':[] if method=='get' else [{'setupToken':[]}], 'responses':{'200':{'description':'Current state; no secrets','content':{'application/json':{'schema':status}}},**{str(c):{'description':d} for c,d in [(400,'Invalid input'),(403,'Disabled, expired, token or Origin rejected'),(409,'Setup already completed'),(429,'Capacity reached'),(500,'Internal failure'),(503,'Database or identity provider unavailable')]}}}
            for code,response in op['responses'].items():
                if code!='200':response['content']={'application/json':{'schema':{'anyOf':[obj({'state':{'type':'string','enum':['disabled','complete','invalid_input','identity_unavailable','setup_failed','unavailable']}}),{'$ref':'#/components/schemas/Error'}]}}}
            if method=='post':
                op['parameters']=[{'name':'Origin','in':'header','required':True,'schema':{'type':'string'}}]
                op['requestBody']={'required':True,'content':{'application/json':{'schema':{'$ref':'#/components/schemas/'+name}}}}
            api['paths'][path][method]=op
        outputs['contracts/schemas/'+name.lower()+'-v1.json']={'$schema':'https://json-schema.org/draft/2020-12/schema','$id':'https://filex.invalid/schemas/'+name.lower()+'-v1.json',**body}
