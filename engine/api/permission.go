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
		find, err := api.Cache.Get(k, &ok)
		if err != nil {
			log.Error("cannot get from cache %s: %v", k, err)
		}
		if find {
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
		if err := api.Cache.SetWithTTL(k, ok, 60*15); err != nil {
			log.Error("cannot SetWithTTL: %s: %v", k, err)
		}
		if !ok {
			log.Error("checkWorkerPermission> actionBuildID:%v runNodeJob.ID:%v", getWorker(ctx).ActionBuildID, runNodeJob.ID)
		}
		return ok
	}
	return true
} */
