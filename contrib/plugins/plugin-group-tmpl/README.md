# group-tmpl
plugin-group-tmpl is an helper to create a deployment file for an application group on mesos/marathon using golang text/template
Once the file is generated, you can simply push the file to mesos/marathon as a group update.

## How to build

Make sure go is installed and properly configured ($GOPATH must be set)

```shell
$ go test ./...
$ go build
```

## How to install

### Install as CDS plugin

As CDS admin:

- login to Web Interface,
- go to Action admin page,
- choose the plugin binary file freshly built
- click on "Add plugin"

## How to use

### Parameters

- **config**, a template marathon config to apply to every application, containing golang text/template variables
- **applications**, a variables file to overwrite the variables in the template file, containing both default variables to have an overall default configuration, and apps variables to be able to configure apps by apps
- **output**, generated file, takes by the default <config>.out or just trimming the .tpl extension

## How it works

Basically, it will takes all the variables defines for every application, apply them on the config file to create an output file containing an array of marathon application.

The application file has many features :
- **id** : the key of the application in the 'apps' section defines a variables named "id"
- **default variables** : it defines all the variables being present for every 'apps'. It can be overwritten by an application variable if redefined.
- **[[ .var ]]** : *(only in the default variables)* golang text/template variables are allowed, excepted the delimiters are [[ ]] instead of {{ }} and the vars must be defined in the apps section
- **...** : *(only in the apps variables)* preprend the default variable

## Examples

Template file
```json
{
        "id": "{{.id}}",
        "image": "{{.image}}",
        "instances": {{.instances}},
        "cpus": {{.cpus}},
        "env": "{{.env}}",
        "mem": {{.mem}}
}

```

Applications file :
```json
{
    "default": {
        "mem":"512",
        "cpus":"0.5",
        "image": "docker.registry/my-awesome-image",
        "env": "APPLICATION=[[.id]]",
        "instances": 1
    },
    "apps": {
        "first": {
        },
        "second": {
            "cpus": "3",
            "mem": 2048
        },
        "third": {
            "env": "...;SHELL=/usr/bin/zsh"
        }
    }
}
```

Then the output file will be 
```json
{
  "apps": [
    {
       "id": "first",
       "image": "docker.registry/my-awesome-image",
       "cpus": 1,
       "instances": 1,
       "env": "APPLICATION=first",
       "mem": 512
     }, {
       "id": "second",
       "image": "docker.registry/my-awesome-image",
       "cpus": 3,
       "instances": 1,
       "env": "APPLICATION=second",
       "mem": 2048
     }, {
       "id": "third",
       "image": "docker.registry/my-awesome-image",
       "cpus": 1,
       "instances": 1,
       "env": "APPLICATION=third,SHELL=/usr/bin/zsh",
       "mem": 512
     }
 ]
}
```