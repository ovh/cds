+++
title = "Write your own hatchery"
weight = 1

[menu.main]
parent = "tutorials"
identifier = "tutorials-hatchery-create"

+++

### Example with a creation of a VSphere hatchery

* First of all you need to create a new package like the other into the hatchery package. Let's call this package vsphere for our example.

* You have to implement the Service interface (see [here](https://github.com/ovh/cds/blob/master/engine/types.go)) in order to configure launch this new hatchery mode via CDS engine CLI.

* Your have to create a Configuration structure composed of the [hatchery.CommonConfiguration](https://godoc.org/github.com/ovh/cds/sdk/hatchery#CommonConfiguration) and the variables you need to access to Vsphere API. You finally have to update the [engine main.go file](https://github.com/ovh/cds/blob/master/engine/main.go) to manage this new service and add and manage the configuration structure as part of the [global configuration](https://github.com/ovh/cds/blob/master/engine/types.go).

* You need to implement the hatchery interface (see [here](https://godoc.org/github.com/ovh/cds/sdk/hatchery#Interface))

* The [Init](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.Init) function is useful to create the struct and to set all of these parameters, for our example it's in this function that we're going to create all our VSphere informations fetched from the VSphere API. (cf [init.go](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.Init)). In this case we create a new vsphere client to request vsphere API, a new finder to fetch all informations about our vsphere host. In fact, all informations that we need to spawn and kill the vms with our workers inside on the vsphere infrastructure. This function is also used to create and register the hatchery on the api via the function in the sdk called [hatchery.Register](https://godoc.org/github.com/ovh/cds/sdk/hatchery#Register). This register will give you the id of your hatchery.

* The [ID](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.ID) function returns the id of the current hatchery that comes from the hatchery client registered with the sdk previously in the [Init](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.Init) function.

* The [ModelType](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.ModelType) function returns the type of the hatchery, in this case it's vsphere type. We can create a constant VSphere inside our [sdk package](https://godoc.org/github.com/ovh/cds/sdk#pkg-constants).

* [Hatchery](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.Hatchery) function returns the hatchery struct initialised previously in the `Init` function.

* [Client](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.Client) function returns the sdk client initialised previously in the `Init` function too.

* [NeedRegistration](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.NeedRegistration) is used to know if your worker model need registration. For example if a user update a worker model you have to rebuild the virtual machine model linked to this worker model. And this function returns true if the worker model was updated after the virtual machine model was created on vsphere. In order to know that for vsphere we have add some metadata on each vms in order to add more custom data as the last creation of this vm for example.

* [CanSpawn](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.CanSpawn) function checks if the hatchery can spawn this model. For example with vsphere with check if there is a memory requirement, if it's the case it returns false because it's not already supported by our vsphere hatchery for now.

* [WorkersStartedByModel](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.WorkersStartedByModel) returns all workers which are running with a worker model. In our vsphere example, in order to know that we register a string metadata called model which tell us the name of our worker model.

* [WorkersStarted](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.WorkersStarted) returns all workers include those which are running to build a vm model or to register a worker model. For example in the vsphere case, in order to count them we all prefix our worker spawned with `worker-`.

* [CanSpawn](https://godoc.org/github.com/ovh/cds/engine/hatchery/vsphere#HatcheryVSphere.CanSpawn) is where the magic happens, it's in this function that you spawn your vms with the right configuration. In fact with vsphere we check if there is a vm model already created for the worker model passed in parameter. If it doesn't we create a vm model with the user data store in worker model infos. Then we spawn a vm with right environment variables created from informations passed in parameters (workerModel, registerOnly, ...) and with a script inside the vm that download the worker binary, execute it and shutdown when all is done. In this function we also check if the worker should be launch to register or to execute steps. When we launch a worker for register it means that the worker is launched and then send all there binary capabilities to the api for this worker model but don't execute any jobs.

* In our vsphere implementation we also launch multiple goroutine to clean and kill workers which seem down or in error. It's a ticker that check all vms state in a periodic way.

### Test

If you want to test that you just have to launch it like that :

```bash
$ engine start hatchery:vsphere --config config.toml
```
