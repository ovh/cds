# CDS Hooks µService

## Introduction

CDS Hooks µService is the component designed to run all workflow hooks.

Following hooks are supported:

- Webhook
- Scheduler

Following will be supported:

- Kafka Listener
- Github, Gitlab, Bitbucket Poller

## Design

All **hooks** are considered as **task**. Tasks are synchronized on startup with workflow hooks from CDS API, and added on the fly when CDS API call this µService.

When a hook is invocated or have to be invocated, we talk about **task execution**.

**Task execution** are managed by an internal scheduler `Service.runScheduler(context.Context)`.
The scheduler is design in three part:

- the task execution processor `Service.dequeueTaskExecutions(context.Context)`: To feed it, we have to push a **task execution key** in the queue `hooks:scheduler:queue`.
- the task execution retry `Service.retryTaskExecutionsRoutine(context.Context)`: Which checks all executions to push in the queue `hooks:scheduler:queue` the not processed task execution
- the task execution cleaner `Service.deleteTaskExecutionsRoutine(context.Context)`: Which removes old task executions.

## Storage

Task list and definitions are stored in the *Cache* (redis or local). The key `hooks:tasks` is a Sorted Set containing tasks UUID sorted by timestamp creation.
Each task is stored as JSON in a key `hooks:tasks:<UUID>`.

When a **task** is or have to be invocated, the **task execution** of the **task** is listed in a Sorted Set (sorted by timestamp of **task execution**): `hooks:tasks:executions:<type>:<UUID>`; this set contains the list of all timestamp on **task execution**.
The detail of an **task execution** is stored as JSON in. The **task execution key** is `hooks:tasks:executions:<type>:<UUID>:<timestamp>`

## API

Following routes are available:

- `GET|POST|PUT|DELETE /webhook/{uuid}` : Routes available for the webhooks. No authentication.

- `POST /task`: Create a new task from a CDS `sdk.WorkflowNodeHook`. Authentication: Header `X_AUTH_HEADER`: `<Service Hash>` in base64
- `GET|PUT|DELETE /task/{uuid}`: Get, Update or Delete a task. Authentication: Header `X_AUTH_HEADER`: `<Service Hash>` in base64
- `GET /task/{uuid}/execution`: Get all task execution. Authentication: Header `X_AUTH_HEADER`: `<Service Hash>` in base64

## Authentication

The µService is run with a `shared.infra` token and register on CDS API; on registration, CDS API gives in response a hash (**service hash**) which must be used to make every call to CDS API. Every 30 seconds, it heartbeats on CDS API.

## How to implement a new type of hook ?

All the specific code for each type of hook (task) is in the `tasks.go` file.

1. Declare a new const for the type
1. Update `hookToTask` function which convert a CDS `sdk.WorkflowNodeHook` to a **task**
1. Update `startTask` function if you have some thing special to do to start or prepare the next task execution
1. Update `stopTask` function if you have some thing special to do to stop a task
1. Update `doTask` function and create a `do<your-stuff>Execution` function with your business