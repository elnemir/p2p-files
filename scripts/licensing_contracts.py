"""Licensing contracts shared by customer verification and the isolated issuer."""
def object_schema(props):
    return {'type':'object','additionalProperties':False,'required':list(props),'properties':props}

def outputs():
    string={'type':'string','minLength':1,'maxLength':128,'pattern':'^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$'}
    uuid={'type':'string','format':'uuid','pattern':'^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'}
    date={'type':'string','format':'date-time','pattern':'Z$'}
    positive={'type':'integer','minimum':1,'maximum':9007199254740991}
    nullable=lambda x: {'anyOf':[x,{'type':'null'}]}
    claims=object_schema({'schemaVersion':{'const':1},'product':{'const':'filex'},'issuer':{'const':'filex-vendor'},'licenseId':uuid,'entitlementId':uuid,'revision':positive,'customerId':string,'installationId':uuid,'kind':{'enum':['subscription','perpetual']},'issuedAt':date,'notBefore':date,'expiresAt':nullable(date),'updatesUntil':date,'supportUntil':date,'limits':object_schema({'maxEnabledUsers':nullable(positive),'maxTransferBytes':nullable(positive)}),'features':{'type':'array','uniqueItems':True,'items':{'enum':['webdav','federated_auth','white_label']}}})
    claims['allOf']=[{'if':{'properties':{'kind':{'const':'subscription'}}},'then':{'properties':{'expiresAt':date}},'else':{'properties':{'expiresAt':{'type':'null'}}}}]
    issue=object_schema({'idempotencyKey':uuid,'claims':claims})
    api={'openapi':'3.1.1','info':{'title':'FileX private license issuer','version':'1.0.0','description':'Isolated vendor service. Named OIDC subjects and MFA required. Customer instances never contact this service.'},'paths':{},'components':{'schemas':{'Claims':claims,'Issue':issue},'securitySchemes':{'session':{'type':'apiKey','in':'cookie','name':'issuerSession'}}},'security':[{'session':[]}]}
    for path,method,summary in [('/issuer/api/session','get','Read the current authorized operator'),('/issuer/api/issuances','get','List immutable issuance history'),('/issuer/api/issuances','post','Approve an idempotent issuance for the isolated signer'),('/issuer/api/issuances/{id}','get','Download an issued license'),('/issuer/api/logout','post','Revoke the current session')]:
        op={'summary':summary,'responses':{'200':{'description':'Success'},'400':{'description':'Invalid input'},'401':{'description':'Authentication required'},'403':{'description':'Role, fresh MFA, Origin or CSRF rejected'},'409':{'description':'Idempotency or revision conflict'},'500':{'description':'Unavailable'}}}
        if path.endswith('{id}'):op['parameters']=[{'name':'id','in':'path','required':True,'schema':uuid}]
        if method=='post' and path.endswith('issuances'):op['requestBody']={'required':True,'content':{'application/json':{'schema':{'$ref':'#/components/schemas/Issue'}}}}
        api['paths'].setdefault(path,{})[method]=op
    session=object_schema({'subject':{'type':'string'},'role':{'enum':['viewer','issuer','security_admin']},'csrfToken':{'type':'string'},'authenticatedAt':date})
    issuance=object_schema({'id':uuid,'state':{'enum':['approved','issued','signing_failed']},'claims':{'$ref':'#/components/schemas/Claims'},'sha256':{'type':'string'},'createdAt':date,'keyId':{'type':'string'}})
    api['components']['schemas'].update({'Session':session,'Issuance':issuance,'Error':object_schema({'code':{'type':'string'}})})
    for path,methods in api['paths'].items():
        for method,op in methods.items():
            if method=='post':op.setdefault('parameters',[]).extend([{'name':name,'in':'header','required':True,'schema':{'type':'string'}} for name in ['Origin','X-CSRF-Token']])
            if path.endswith('{id}'):
                op['responses']['200']['content']={'application/octet-stream':{'schema':{'type':'string','description':'Compact JWS license artifact'}}}
            else:
                schema={'$ref':'#/components/schemas/Session'} if path.endswith('session') else object_schema({'loggedOut':{'const':True}}) if path.endswith('logout') else {'type':'array','items':{'$ref':'#/components/schemas/Issuance'}} if method=='get' else {'$ref':'#/components/schemas/Issuance'}
                op['responses']['200']['content']={'application/json':{'schema':schema}}
            for code,response in op['responses'].items():
                if code!='200':response['content']={'application/json':{'schema':{'$ref':'#/components/schemas/Error'}}}
    for route in ['login','callback']:
        api['paths']['/issuer/auth/'+route]={'get':{'summary':'Start OIDC MFA authorization' if route=='login' else 'Consume a bound OIDC authorization response','security':[],'responses':{'302' if route=='login' else '303':{'description':'Redirect'},'401':{'description':'Invalid identity response'},'403':{'description':'MFA or subject rejected'},'429':{'description':'Flow capacity reached'},'503':{'description':'Unavailable'}}}}
    api['paths']['/issuer/auth/callback']['get']['parameters']=[{'name':name,'in':'query','required':True,'schema':{'type':'string'}} for name in ['state','code']]
    event=object_schema({'issuanceId':uuid,'completedAt':nullable(date)})
    asyncapi={'asyncapi':'3.0.0','info':{'title':'FileX issuer durable signing queue','version':'1.0.0'},'channels':{'signing':{'address':'issuer_outbox','description':'PostgreSQL durable queue. A row is approved atomically with its audit event; signer uses FOR UPDATE SKIP LOCKED. No external broker.','messages':{'issuance':{'payload':event}}}},'operations':{'sign':{'action':'receive','channel':{'$ref':'#/channels/signing'}}}}
    return {'contracts/schemas/license-v1.json':{'$schema':'https://json-schema.org/draft/2020-12/schema','$id':'https://filex.invalid/schemas/license-v1.json',**claims},'contracts/issuer-openapi.json':api,'contracts/issuer-asyncapi.json':asyncapi}
