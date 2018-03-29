+++
title = "First Workflow from repository"
weight = 1
aliases = [
    "/gettingstarted"
]
+++

## Prerequisites

 * Have an account on your CDS instance
 * Have a CDS project and a repository manager has been set up on your CDS Instance!  **[Repository manager]({{< relref "../hosting/repositories-manager" >}})**
 * Have cdsctl command line [Download](https://github.com/ovh/cds/releases)
 
## To get started with CDS

 * In a terminal, go into your git repository and login with cdsctl
 
```
    cd <path-to-repo>
    cdsctl login -H <cds-api-url> -u <username> -p <password>
```

 * Create your first workflow
 
```
    cdsctl workflow init
```

This will ask you to choose:
 * The CDS project
 * The repository manager where your application code is hosted
 * Select 'n' for the question: 'Do you want to reuse an existing pipeline ?'
 * Type your pipeline name
 
 ```
 >cdsctl workflow init
 Choose the CDS project
 	FIRSTPROJECT - MyFirstProject [1]
 Your choice [1-1]: 1
 Initializing workflow from sguiheux/cdsdemo (git@github.com:sguiheux/cdsdemo.git)...
 Choose the repository manager
 	github [1]
 Your choice [1-1]: 1
 application FIRSTPROJECT/cdsdemo (sguiheux/cdsdemo) found in CDS
 Do you want to reuse an existing pipeline ? [y/n]: n
 Enter your pipeline name : MyPipeline
 File .cds/cdsdemo.yml created
 File .cds/MyPipeline.pip.yml created
 Pushing workflow to CDS...
 	Pipeline MyPipeline successfully created
 	Workflow cdsdemo has been updated
 Now you can run: git add .cds/ && git commit -s -m "chore: init CDS workflow files"
 You should consider add the following keys in githubpgp
 -----BEGIN PGP PUBLIC KEY BLOCK-----
 
 xsBNBFqn6wYBCAC3nVRHO/QxBZqvD27jm3f8KhSBA2cxccHSoFQIRykkVy6kLzth
 VV2fxcEOiI9frKBxPQUKeTKQPBkbvqR9/JrP+h0opmLTQ9mQ4l5ax1K7fWazTUlR
 le/fOljJtkjEs9GnsSe348CDc00aN5giQcL6NRyM0IbmYDPo/bcTXRZa/zJYnFAK
 V11AAVLvjokDtA2vFDce6sqaPsu/y4M1tm2vPhef1kvJb3W4kH5soEGem5apKZ81
 kvmmfjxVUFEUZKPGWZEIQli1RP8mTLTi+3B6C7klMkId8tyMo7HD/GGTwHgM3GN1
 /7wTr7SUmIzyg1lTMseAugorIF/2MRqzmWbBABEBAAHNLmFwcC1wZ3AtZ2l0aHVi
 IChhcHAtcGdwLWdpdGh1YikgPGNkc0Bsb2NhaG9zdD7CwGkEEwEIAB0FAlqn6wYJ
 EJp/Gt0iarUSAhsDAhkBAwsJAgIVCgAAtqYIABMo6JFExLVu3Oyl58ouqhfcv7Qr
 VmiwT6rTcvOSAREj/7NB0BNm0gUyhOIvbdMjzSlhl2x6SLlgE1ZankCp+cl5d9GU
 QAWKGs044nntbsUpkVRD/TyJocv2kX88HkarfB11fDGUOfkAEB6cXRMZ6bxgKDw7
 EhwG2qewFJ3bg2713Dpc30ViT0DtuCDF0gUO8eJAIEplu6IT9lmvg44d7+IErkGi
 +Q1uwybtMi85vdb2xgTORjoRACp8O1UVnygvJkjC0oirOghBxB9bMo6ZQDDrVwij
 QASpcijkSlEp/thSGoexNYNztn9w/WEeFHGrk3FOaljsTwH8+6OnqxVm/ubOwE0E
 WqfrBgEIANx9wDczcdUFrAnCoIsncSGUkEtuK9isogKqjkt/USFX4DNv6GMYyqE/
 MUlWrSOHucTRlipXiaTJgPIuICTZPj90NNpf4CABYAJISM9+nwIRAfH8pVyqqexR
 yBmX4QNidi71MdytGaGEyqc4VAeqPyhAFtQ+ON89QnAQCTM2hMQVR7fibUlrn57v
 +D1NjU+e5Ugh56cD5TeknWKqNWV2UC0TWh5Wogbw3NKEAwuc1rqRxODFbRXLzNgL
 5DZFpxvCbFCCsBf7UzuxM8JNqCek2lXN3iHyup47EcPPm0AL3avoevVkoctCYz06
 XOnBGca28SW0SOTIAwtj0z4OBeNjAmInEQEAAcLAXwQYAQgAEwUCWqfrBgkQmn8a
 3SJqtRICGwwAAOt1CABBvmQa9KLK4JdwyYYlfqJW/VASgqw29AJe72pgkGspmo2+
 Jx5wITXMEdgXhcsVRP/3JNfO9NyOhYSp5nSA2/MB02WuPGeug9n69A6EUYJAbnh4
 vhhXY5N6iTymBSmtjZXL/48B5OvG4PLHBflGHUxFcWJqlmq786d0GtqjKUBuD58g
 qNjUpv7Su38WmE6rBGYzpLDpf0X+dQoakZQwRHpRGEdzdlPXnk9UXv783T54sgt0
 WmjyDAHgCf/bPAV1KOOrqmxcIgTQN2+ZWdp9tYt73JV+OIe82VoYhNs+39kBVkgH
 JDrYDf+Fii5pD3fgXlDWqFTwvdSz21OruCg16Ayx
 =lO/4
 -----END PGP PUBLIC KEY BLOCK-----
 
 ssh
 ssh-rsa AAAAB3NzaC1yc2EAA30AADAQABAA06ACAQDyUXiY45Z9Bai7nsj8Tk2olZIwaDhYlBjw60TOlNLWVSvzNS5K+Rps8AA1985A7pcS1tbWtfC2xNMjRN7NznVatioHXgozLTQ/EhKOuEevOp4mCCuebUc62m/14pGKsCN/ikHb6Ca/rf0+NOKt+UUYAOvyVt7FM3NydyT8VrPZWonxULzOIDcPpyZPfbnpuooCZjK2IuaU1pzPxDgszks77bkPePKujhp17Ckfzi+Ke3SgHGr9399UgY8dD6wqvRd+xNQA+EUQNa9SGg5MJ4LgTlqiE+0s/qg7pVUtCLqTo6fSUK0oumkpClmsdwmgnBtfG+5Belli3sMJUHOdw9fQpKUYITQ6jFAaciVzXpdt9j0ImQytz8nP5cd4lpPuv5fHNbx39G7KqdraVsqe3I9Y8RSf2kCVvRON9TthRleYHoxukztsSuxcxeZ0GtaIbIauYymrAvRrAV1harOwSrFThY6sTWyofpZnKesG6S7omIGn5ZjZDtT2p5tiGnZh3gZlS2sHLALyoShUjHxgcOd2h4CMuC1JN63t94rXWfbTYH+eraAhufmnayLC8p4UM/lc0syQBy1aKywR+acScICww20xEt8SG3D5rAlJD5d2EPhJaEzGS0NC6apiaQ2CNlvaceWFoEDbkXj0TL4M0iX42va5Ry1jvo8IPwJ9MoAXrVlowyw==
  app-ssh-github@cds
```

 * CDS generated 3 files for you :
 
   One for your application
   
   ```
   version: v1.0
   name: cdsdemo
   vcs_server: github
   repo: sguiheux/cdsdemo
   keys:
     app-pgp-github:
       type: pgp
       value: b59c70ed26bb4948994927448d506d1d
   vcs_branch: '{{.git.branch}}'
   vcs_default_branch: master
   vcs_pgp_key: app-pgp-github
   ```
   
   One for the pipeline
   
   ```
   version: v1.0
   name: MyPipeline
   jobs:
   - job: First job
     steps:
     - checkout: '{{.cds.workspace}}'
     requirements:
     - binary: git
   ```
   
   One for the workflow
   
   ```
    name: cdsdemo
    version: v1.0
    pipeline: MyPipeline
    payload:
      git.author: ""
      git.branch: ""
      git.hash: ""
      git.hash.before: ""
      git.message: ""
      git.repository: sguiheux/cdsdemo
    application: cdsdemo
    pipeline_hooks:
    - type: RepositoryWebHook

   ```     

 * Commit and push cds files
 ```
 git add .cds/ && git commit -s -m "chore: init CDS workflow files"
 ``` 
 
 * CDS triggered your workflow from your git push. You can track execution
 ```
>cdsctl workflow status --track
cdsdemo [b226e84 | steven.guiheux] #1.0 ➤ ✓ MyPipeline     
 ```

 * On CDS UI
 
![Run1](/images/getting.started.run1.png)

## Update pipeline to execute a mvn package

 * Edit the pipeline file to add 
 
```
version: v1.0
name: MyPipeline
jobs:
- job: First job
  steps:
  - checkout: '{{.cds.workspace}}'
  - script : mvn package
  - artifactUpload : target/*.jar
  requirements:
  - binary: mvn
  - binary: git

```

```
>cdsctl workflow status --track
cdsdemo [d98bd14 | steven.guiheux] #2.0 ➤ ✓ MyPipeline
```

 * On CDS UI
 
![Run2](/images/getting.started.run2.png)
