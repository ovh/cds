+++
title = "Webhook"
weight = 2

+++


On a Root Pipeline, you can add a "Webhook". Click on the created icon to get the WebHook URL.

In order to trigger this one you just have to make a HTTP call on the given url with the selected method. If the selected method is `POST` you can also send a payload from your workflow inside the request body or if you use `GET` method you can write your payload using query parameters.

![Webhook](/images/workflows.design.hooks.webhook.gif)
