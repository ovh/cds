---
title: "Development"
weight: 20
toc: true
prev: "/tatwebui/cdsview"

---

## Introduction

A view in Tat WebUI is a bower's plugin. You can develop your own view to display message or add specific action on messages.

## Steps
### Requirements
```
#Install NodeJs [https://nodejs.org/]
#Install Bower
npm install -g bower

#Install Grunt
npm install -g grunt-cli
```

### Tatwebui bootstrap

```
mkdir -p $HOME/src/github.com/ovh #you can used what you want, it's just for example
git clone https://github.com/ovh/tatwebui.git
cd tatwebui/client
touch plugin.tpl.json custom.plugin.tpl.json src/assets/config.json
cd ../server
touch app/config.json
```


### Tatwebui configuration
In file client/src/assets/config.json, add theses lines
```
{
  "backend": {
    "scheme": "https",
    "host": "url-tat-engine",
    "port": 443,
    "autologin": false
  },
  "releaseview": {
    "tracker": "https://github.com/ovh/tat/issues/",
    "keyword": "github"
  },
  "help": {
    "signup": ["l'aide ici"]
  },
  "links": {
    "home": [{
      "caption": "label home 1",
      "url": "http://tat-engine.tat.tat.home1"
    }, {
      "caption": "label home 2",
      "url": "http://tat-engine.tat.tat.home2"
    }],
    "menu": [{
      "caption": "label 1",
      "url": "http://tat-engine.tat.tat.label1"
    }, {
      "caption": "label 2",
      "url": "http://tat-engine.tat.tat.label2"
    }]
  },
  "viewconfigs": {
    "standardview-list": {
      "filters": {
        "placeholder": {
          "text": "text",
          "label": "open,doing",
          "andLabel": "open,doing",
          "notLabel": "done",
          "tag": "PCC,STOCKAGE",
          "andTag": "PCC,STOCKAGE",
          "notTag": "PCC,STOCKAGE"
        }
      }
    }
  }
}

```

In file server/app/config.json, add theses lines
This file is not really used in development, but it's checked by make target.

```
{
 "proxy": {
   "tatEngine": {
     "scheme": "http",
     "host": "ip_or_domain_of_proxy",
     "port": 8080,
     "sslInsecureSkipVerify": false
   },
   "listen_port": 8001
 },
 "filesystem": {
     "listen_port": 8000
 },
 "process": {
   "nb_forks": "2"
 }
}
```

### Plugins configuration
In file plugin.tpl.json, add theses lines
```

{
  "dependencies": {
    "tatwebui-plugin-standardview": "git+https://github.com/ovh/tatwebui-plugin-standardview.git",
    "tatwebui-plugin-notificationsview": "git+https://github.com/ovh/tatwebui-plugin-notificationsview.git",
    "tatwebui-plugin-cdsview": "git+https://github.com/ovh/tatwebui-plugin-cdsview.git",
    "tatwebui-plugin-monitoringview": "git+https://github.com/ovh/tatwebui-plugin-monitoringview.git",
    "tatwebui-plugin-pastatview": "git+https://github.com/ovh/tatwebui-plugin-pastatview.git",
    "tatwebui-plugin-dashingview": "git+https://github.com/ovh/tatwebui-plugin-dashingview.git",
    "tatwebui-plugin-releaseview": "git+https://github.com/ovh/tatwebui-plugin-releaseview.git"
  }
}
```

### First Run without additional plugin

```
cd $HOME/src/github.com/ovh/tatwebui
make release && make run
```


Then go to http://localhost:8000/ , check if it's ok.
Run in Dev Mode
```
cd $HOME/src/github.com/ovh/tatwebui
make devclient
```

Then go to http://localhost:9000/ and use your tat_username and tat_password to log in.

## Bootstrap your plugin

```
cd $HOME/src/
git clone https://github.com/ovh/tatwebui-plugin-standardview.git
mv tatwebui-plugin-standardview tatwebui-plugin-yourpluginview
cd tatwebui-plugin-yourpluginview
# then rename all files *standardview* to *yourpluginview*
# then rename all code *Standardview* to *Yourpluginview*, *standardView* to *yourpluginView* -> it's case sensitive !
# adjust files bower.json file and yourpluginview-list.route.js
```


### Tatwebui Plugin Configuration for dev

In file custom.plugin.tpl.json, add theses lines
```

{
  "dependencies": {
    "your-plugin-view": "git+https://.../you-plugin-view.git"
  }
}
```

cd $HOME/src/github.com/ovh/tatwebui/client/bower_components
ln -s $HOME/you-plugin-view tatwebui-plugin-yourpluginview
cd $HOME/src/github.com/ovh/tatwebui
make devclient
```

Then go to http://localhost:9000/ and check if your view is available on top right
