+++
title = "Scheduler"
weight = 1

+++

On a Root Pipeline, you can add a "Hook Scheduler". This kind of hook is useful when you want to launch a workflow periodically (for example each day at 1AM). You can use the [Crontab Expression Format](https://github.com/gorhill/cronexpr#implementation) to configure your scheduler's period. You can also configure a specific payload for your scheduler.

![Scheduler](/images/workflows.design.hooks.scheduler.gif)
