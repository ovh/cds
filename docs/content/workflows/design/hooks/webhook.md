+++
title = "Webhook"
weight = 2

+++


On a Root Pipeline, you can add a "Webhook". Click on the created icon to get the WebHook URL.

In order to trigger this one you just have to make a curl on the given url with the selected method. If the selected method is `POST` you can also send a payload from your workflow inside the request body.

![Webhook](/images/workflows.design.hooks.webhook.gif)
