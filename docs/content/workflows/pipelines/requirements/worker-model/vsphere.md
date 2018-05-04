+++
title = "VSphere Worker Model"
weight = 4

+++

CDS build using VSphere infrastructure

## Add VSphere worker model

We need to define a VSphere worker model to have VSphere hatchery booting workers.

We will create a worker model called debian8-docker:

 * On Debian 8
 * With docker ready to use
 * Git installed


 You need to configure:

   * The image is the name of your virtual machine that you have created before on your host to clone (See [Advanced]({{< relref "hatchery/vsphere.md" >}}))
   * If you aren't an administrator you have to choose a configuration pattern in order to fill pre command, worker command and post command with a [pattern that an administrator have already fill for you]({{< relref "workflows/TODO_PATTERNS" >}}).
   * If you are an administrator :
     * pre worker command: all scripts that need to be run before execute the worker binary (for example: set the right environment variables, install curl and other tools you need like docker, ...)
     * main worker command: the command launched to run the worker with right flags thanks to the interpolate variables that CDS fill for you [(more informations click here)]({{< relref "workflows/TODO_VARIABLES" >}}).
     * post worker command: the command launched after the execution of your worker. If you need to clean something and then shutdown the VM.

 Via UI (inside settings section --> worker models):

 For example:

 ![Worker Model UI VSphere](/images/worker_model_vsphere.png)

 Or via cli with a yaml file:

 ```bash
 $ cds worker model import my_worker_model.yml
 ```


 ```yaml
 name: testvsphere
 type: vsphere
 description: "my worker model"
 group: shared.infra
 image: debian8
 pre_cmd: |
   #!/bin/bash
   set +e
   export CDS_FROM_WORKER_IMAGE={{.FromWorkerImage}}
   export CDS_SINGLE_USE=1
   export CDS_FORCE_EXIT=1
   export CDS_API={{.API}}
   export CDS_TOKEN={{.Key}}
   export CDS_NAME={{.Name}}
   export CDS_MODEL={{.Model}}
   export CDS_HATCHERY={{.Hatchery}}
   export CDS_HATCHERY_NAME={{.HatcheryName}}
   export CDS_BOOKED_PB_JOB_ID={{.PipelineBuildJobID}}
   export CDS_BOOKED_WORKFLOW_JOB_ID={{.WorkflowJobID}}
   export CDS_TTL={{.TTL}}
   export CDS_INSECURE={{.HTTPInsecure}}

   # Basic build binaries
   cd $HOME
   apt-get -y --force-yes update >> /tmp/user_data 2>&1
   apt-get -y --force-yes install curl git >> /tmp/user_data 2>&1
   apt-get -y --force-yes install binutils >> /tmp/user_data 2>&1
   # Docker installation (FOR DEBIAN)
   if [[ "x{{.FromWorkerImage}}" = "xtrue" ]]; then
     echo "$(date) - CDS_FROM_WORKER_IMAGE == true - no install docker required "
   else
     # Install docker
     apt-get install -y --force-yes apt-transport-https ca-certificates >> /tmp/user_data 2>&1
     apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
     mkdir -p /etc/apt/sources.list.d
     sh -c "echo deb https://apt.dockerproject.org/repo debian-jessie main > /etc/apt/sources.list.d/docker.list"
     apt-get -y --force-yes update >> /tmp/user_data 2>&1
     apt-cache policy docker-engine >> /tmp/user_data 2>&1
     apt-get install -y --force-yes docker-engine >> /tmp/user_data 2>&1
     service docker start >> /tmp/user_data 2>&1
     # Non-root access
     groupadd docker >> /tmp/user_data 2>&1
     gpasswd -a ${USER} docker >> /tmp/user_data 2>&1
     service docker restart >> /tmp/user_data 2>&1
   fi;
   curl -L "{{.API}}/download/worker/linux/$(uname -m)" -o worker --retry 10 --retry-max-time 120 -C - >> /tmp/user_data 2>&1
   chmod +x worker

 cmd: "PATH=$PATH ./worker"

 post_cmd: sudo shutdown -h now

 ```
