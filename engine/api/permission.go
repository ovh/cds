package api

/*
func (api *API) checkWorkerPermission(ctx context.Context, db gorp.SqlExecutor, rc *service.HandlerConfig, routeVar map[string]string) bool {
	if getWorker(ctx) == nil {
		log.Error("checkWorkerPermission> no worker in ctx")
		return false
	}

	idS, ok := routeVar["permID"]
	if !ok {
		return true
	}

	id, err := strconv.ParseInt(idS, 10, 64)
	if err != nil {
		log.Error("checkWorkerPermission> Unable to parse permID:%s err:%v", idS, err)
		return false
	}

	//IF it is POSTEXECUTE, it means that the job is must be taken by the worker
	if rc.Options["isExecution"] == "true" {
		k := cache.Key("workers", getWorker(ctx).ID, "perm", idS)
		if api.Cache.Get(k, &ok) {
			if ok {
				return ok
			}
		}

		runNodeJob, err := workflow.LoadNodeJobRun(db, api.Cache, id)
		if err != nil {
			log.Error("checkWorkerPermission> Unable to load job %d err:%v", id, err)
			return false
		}

		ok = runNodeJob.ID == getWorker(ctx).ActionBuildID
		api.Cache.SetWithTTL(k, ok, 60*15)
		if !ok {
			log.Error("checkWorkerPermission> actionBuildID:%v runNodeJob.ID:%v", getWorker(ctx).ActionBuildID, runNodeJob.ID)
		}
		return ok
	}
	return true
} */
