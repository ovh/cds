package main

const schema = `
{
  "$schema": "http://json-schema.org/schema#",
  "additionalProperties": false,
  "definitions": {
    "pathType": {
      "type": "string",
      "pattern": "^(\\/?((\\.{2})|([a-z0-9][a-z0-9\\-.]*[a-z0-9]+)|([a-z0-9]*))($|\\/))+$",
      "minLength": 1
    }
  },
  "not": {
    "allOf": [
      {
        "required": [
          "cmd"
        ]
      },
      {
        "required": [
          "args"
        ]
      }
    ]
  },
  "properties": {
    "acceptedResourceRoles": {
      "items": {
        "type": "string"
      },
      "type": "array",
      "description": "Optional. A list of resource roles. Marathon considers only resource offers with roles in this list for launching tasks of this app. If you do not specify this, Marathon considers all resource offers with roles that have been configured by the --default_accepted_resource_roles command line flag. If no --default_accepted_resource_roles was given on startup, Marathon considers all resource offers. To register Marathon for a role, you need to specify the --mesos_role command line flag on startup. If you want to assign all resources of a slave to a role, you can use the --default_role argument when starting up the slave. If you need a more fine-grained configuration, you can use the --resources argument to specify resource shares per role. The Mesos master needs to be started with --roles followed by a comma-separated list of all roles you want to use across your cluster. See [the Mesos command line documentation](http://mesos.apache.org/documentation/latest/configuration/) for details."
    },
    "args": {
      "items": {
        "type": "string"
      },
      "type": "array",
      "description": "An array of strings that represents an alternative mode of specifying the command to run. This was motivated by safe usage of containerizer features like a custom Docker ENTRYPOINT. This args field may be used in place of cmd even when using the default command executor. This change mirrors API and semantics changes in the Mesos CommandInfo protobuf message starting with version 0.20.0.  Either cmd or args must be supplied. It is invalid to supply both cmd and args in the same app."
    },
    "backoffFactor": {
      "minimum": 1.0,
      "type": "number",
      "description": "Configures exponential backoff behavior when launching potentially sick apps. This prevents sandboxes associated with consecutively failing tasks from filling up the hard disk on Mesos slaves. The backoff period is multiplied by the factor for each consecutive failure until it reaches maxLaunchDelaySeconds. This applies also to tasks that are killed due to failing too many health checks."
    },
    "backoffSeconds": {
      "description": "Configures exponential backoff behavior when launching potentially sick apps. This prevents sandboxes associated with consecutively failing tasks from filling up the hard disk on Mesos slaves. The backoff period is multiplied by the factor for each consecutive failure until it reaches maxLaunchDelaySeconds. This applies also to tasks that are killed due to failing too many health checks.",
      "minimum": 0,
      "type": "integer"
    },
    "cmd": {
      "description": "The command that is executed.  This value is wrapped by Mesos via /bin/sh -c ${app.cmd}.  Either cmd or args must be supplied. It is invalid to supply both cmd and args in the same app.",
      "type": "string",
      "minLength": 1
    },
    "constraints": {
      "type": "array",
      "items": {
        "type": "array",
        "items": [
          {
            "type": "string"
          },
          {
            "type": "string",
            "enum": ["UNIQUE", "CLUSTER", "GROUP_BY", "LIKE", "UNLIKE"]
          },
          {
            "type": "string"
          }
        ],
        "minItems": 2,
        "additionalItems": false
      },
      "description": "Valid constraint operators are one of UNIQUE, CLUSTER, GROUP_BY, LIKE, UNLIKE."
    },
    "container": {
      "additionalProperties": false,
      "properties": {
        "docker": {
          "additionalProperties": false,
          "properties": {
            "forcePullImage": {
              "type": "boolean",
              "description": "The container will be pulled, regardless if it is already available on the local system."
            },
            "image": {
              "type": "string",
              "minLength": 1,
              "description": "The name of the docker image to use."
            },
            "network": {
              "type": "string",
              "description": "The networking mode, this container should operate in. One of BRIDGED|HOST|NONE",
              "enum": ["BRIDGE", "HOST", "NONE"]
            },
            "parameters": {
              "type": "array",
              "description": "Allowing arbitrary parameters to be passed to docker CLI. Note that anything passed to this field is not guaranteed to be supported moving forward, as we might move away from the docker CLI.",
              "items": {
                "type": "object",
                "additionalProperties": false,
                "properties": {
                  "key": {
                    "type": "string",
                    "minLength": 1,
                    "description": "Key of this parameter"
                  },
                  "value": {
                    "type": "string",
                    "description": "Value of this parameter"
                  }
                },
                "required": [ "key", "value" ]
              }
            },
            "portMappings": {
              "type": "array",
              "items": {
                "type": "object",
                "additionalProperties": false,
                "properties": {
                  "containerPort": {
                    "type": "integer",
                    "description": "Refers to the port the application listens to inside of the container. It is optional and now defaults to 0, in which case Marathon assigns the container port the same value as the assigned hostPort. This is especially useful for apps that advertise the port they are listening on to the outside world for P2P communication. Without containerPort: 0 they would erroneously advertise their private container port which is usually not the same as the externally visible host port.",
                    "maximum": 65535,
                    "minimum": 0
                  },
                  "hostPort": {
                    "type": "integer",
                    "description": "Retains the traditional meaning in Marathon, which is a random port from the range included in the Mesos resource offer. The resulting host ports for each task are exposed via the task details in the REST API and the Marathon web UI. hostPort is optional and defaults to 0.",
                    "maximum": 65535,
                    "minimum": 0
                  },
                  "labels": {
                    "type": "object",
                    "description": "This can be used to add metadata to be interpreted by external applications such as firewalls.",
                    "additionalProperties": {
                      "type": "string"
                    }
                  },
                  "name": {
                    "type": "string",
                    "description": "Name of the service hosted on this port. If provided, it must be unique over all port mappings.",
                    "pattern": "^[a-z][a-z0-9-]*$"
                  },
                  "protocol": {
                    "type": "string",
                    "enum": ["tcp", "udp", "udp,tcp"],
                    "description": "Protocol of the port (one of [&#39;tcp&#39;, &#39;udp&#39;] or &#39;udp,tcp&#39; for both). Defaults to &#39;tcp&#39;."
                  },
                  "servicePort": {
                    "type": "integer",
                    "description": "Is a helper port intended for doing service discovery using a well-known port per service. The assigned servicePort value is not used/interpreted by Marathon itself but supposed to used by load balancer infrastructure. See Service Discovery Load Balancing doc page. The servicePort parameter is optional and defaults to 0. Like hostPort, If the value is 0, a random port will be assigned. If a servicePort value is assigned by Marathon then Marathon guarantees that its value is unique across the cluster. The values for random service ports are in the range [local_port_min, local_port_max] where local_port_min and local_port_max are command line options with default values of 10000 and 20000, respectively.",
                    "maximum": 65535,
                    "minimum": 0
                  }
                }
              }
            },
            "privileged": {
              "type": "boolean",
              "description": "Run this docker image in privileged mode."
            }
          },
          "required": [
            "image"
          ],
          "type": "object"
        },
        "type": {
          "type": "string",
          "description": "Supported container types at the moment are DOCKER and MESOS.",
          "enum": ["MESOS", "DOCKER"]
        },
        "volumes": {
          "items": {
            "additionalProperties": false,
            "properties": {
              "containerPath": {
                "type": "string",
                "description": "The path of the volume in the container",
                "minLength": 1
              },
              "hostPath": {
                "type": "string",
                "description": "The path of the volume on the host",
                "minLength": 1
              },
              "persistent": {
                "additionalProperties": false,
                "properties": {
                  "size": {
                    "type": "integer",
                    "description": "The size of the persistent volume in MiB",
                    "minimum": 0
                  }
                },
                "type": "object"
              },
              "external": {
                "additionalProperties": false,
                "properties": {
                  "size": {
                    "type": "integer",
                    "description": "The size of the external volume in MiB",
                    "minimum": 0
                  },
                  "name": {
                    "type": "string",
                    "description": "The name of the volume"
                  },
                  "provider": {
                    "type": "string",
                    "description": "The name of the volume provider"
                  },
                  "options": {
                    "type": "object",
                    "description": "Provider-specific volume configuration options"
                  }
                },
                "type": "object"
              },
              "mode": {
                "type": "string",
                "description": "Possible values are RO for ReadOnly and RW for Read/Write",
                "enum": ["RO", "RW"]
              }
            },
            "type": "object"
          },
          "type": "array"
        }
      },
      "type": "object"
    },
    "cpus": {
      "type": "number",
      "description": "The number of CPU shares this application needs per instance. This number does not have to be integer, but can be a fraction.",
      "minimum": 0
    },
    "dependencies": {
      "type": "array",
      "description": "A list of services upon which this application depends. An order is derived from the dependencies for performing start/stop and upgrade of the application. For example, an application /a relies on the services /b which itself relies on /c. To start all 3 applications, first /c is started than /b than /a.",
      "items": {
        "$ref": "#/definitions/pathType"
      }
    },
    "disk": {
      "type": "number",
      "description": "How much disk space is needed for this application. This number does not have to be an integer, but can be a fraction.",
      "minimum": 0
    },
    "env": {
      "patternProperties": {
        ".*": {
          "type": "string"
        }
      },
      "type": "object"
    },
    "executor": {
      "type": "string",
      "description": "The executor to use to launch this application. Different executors are available. The simplest one (and the default if none is given) is //cmd, which takes the cmd and executes that on the shell level.",
      "pattern": "^(|\\/\\/cmd|\\/?[^\\/]+(\\/[^\\/]+)*)$"
    },
    "fetch": {
      "type": "array",
      "description": "Provided URIs are passed to Mesos fetcher module and resolved in runtime.",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "uri": {
            "type": "string",
            "description": "URI to be fetched by Mesos fetcher module"
          },
          "executable": {
            "type": "boolean",
            "description": "Set fetched artifact as executable"
          },
          "extract": {
            "type": "boolean",
            "description": "Extract fetched artifact if supported by Mesos fetcher module"
          },
          "cache": {
            "type": "boolean",
            "description": "Cache fetched artifact if supported by Mesos fetcher module"
          }
        },
        "required": [ "uri" ]
      }
    },
    "healthChecks": {
      "items": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "command": {
            "type": "object",
            "items": {
              "additionalProperties": false,
              "properties": {
                "value": {
                  "type": "string",
                  "description": "The health check command to execute."
                }
              }
            }
          },
          "gracePeriodSeconds": {
            "type": "integer",
            "description": "Health check failures are ignored within this number of seconds of the task being started or until the task becomes healthy for the first time.",
            "minimum": 0
          },
          "ignoreHttp1xx": {
            "type": "boolean",
            "description": "Ignore HTTP 1xx responses."
          },
          "intervalSeconds": {
            "type": "integer",
            "description": "Number of seconds to wait between health checks.",
            "minimum": 0
          },
          "maxConsecutiveFailures": {
            "type": "integer",
            "description": "Number of consecutive health check failures after which the unhealthy task should be killed.",
            "minimum": 0
          },
          "path": {
            "type": "string",
            "description": "Path to endpoint exposed by the task that will provide health status. Example: /path/to/health. Note: only used if protocol == HTTP[S]."
          },
          "port": {
            "type": "integer",
            "description": "The specific port to connect to. In case of dynamic ports, see portIndex.",
            "maximum": 65535,
            "minimum": 0
          },
          "portIndex": {
            "type": "integer",
            "description": "Index in this app&#39;s ports array to be used for health requests. An index is used so the app can use random ports, like [0, 0, 0] for example, and tasks could be started with port environment variables like $PORT1.",
            "minimum": 0
          },
          "protocol": {
            "type": "string",
            "description": "Protocol of the requests to be performed. One of HTTP, HTTPS, TCP or COMMAND.",
            "enum": ["HTTP", "HTTPS", "TCP", "COMMAND"]
          },
          "timeoutSeconds": {
            "type": "integer",
            "description": "Number of seconds after which a health check is considered a failure regardless of the response.",
            "minimum": 0
          }
        }
      },
      "type": "array"
    },
    "id": {
      "$ref": "#/definitions/pathType",
      "description": "Unique identifier for the app consisting of a series of names separated by slashes. Each name must be at least 1 character and may only contain digits (0-9), dashes (-), dots (.), and lowercase letters (a-z). The name may not begin or end with a dash."
    },
    "instances": {
      "type": "integer",
      "description": "The number of instances of this application to start. Please note: this number can be changed any time as needed to scale the application.",
      "minimum": 0
    },
    "labels": {
      "type": "object",
      "description": "Attaching metadata to apps can be useful to expose additional information to other services, so we added the ability to place labels on apps (for example, you could label apps staging and production to mark services by their position in the pipeline).",
      "additionalProperties": {
        "type": "string"
      }
    },
    "maxLaunchDelaySeconds": {
      "type": "integer",
      "description": "Configures exponential backoff behavior when launching potentially sick apps. This prevents sandboxes associated with consecutively failing tasks from filling up the hard disk on Mesos slaves. The backoff period is multiplied by the factor for each consecutive failure until it reaches maxLaunchDelaySeconds. This applies also to tasks that are killed due to failing too many health checks.",
      "minimum": 0
    },
    "mem": {
      "type": "number",
      "description": "The amount of memory in MB that is needed for the application per instance.",
      "minimum": 0
    },
    "ipAddress": {
      "type": "object",
      "description": "If an application definition includes the &#39;ipAddress&#39; field, then Marathon will request a per-task IP from Mesos. A separate ports/portMappings configuration is then disallowed.",
      "properties": {
        "discovery": {
          "type": "object",
          "description": "Information useful for service discovery.",
          "properties": {
            "ports": {
              "type": "array",
              "description": "Array of objects describing the ports exposed by each task.",
              "items": {
                "type": "object",
                "description": "Port",
                "properties": {
                  "number": {
                    "maximum": 65535,
                    "minimum": 0,
                    "type": "integer",
                    "description": "The port number."
                  },
                  "name": {
                    "type": "string",
                    "description": "Name of the port.",
                    "pattern": "^[a-z][a-z0-9-]*$"
                  },
                  "protocol": {
                    "enum": ["tcp", "udp"],
                    "description": "Protocol of the port (one of [&#39;tcp&#39;, &#39;udp&#39;])."
                  }
                }
              }
            }
          }
        },
        "groups": {
          "type": "array",
          "description": "Array of network groups. One IP address can belong to zero or more network groups. The idea is that containers can only reach containers with which they share at least one network group.",
          "items": {
            "type": "string",
            "description": "The name of the network group."
          }
        },
        "labels": {
          "type": "object",
          "description": "Key value pair for meta data on that network interface.",
          "properties": {},
          "additionalProperties": true
        }
      }
    },
    "ports": {
      "type": "array",
      "description": "An array of required port resources on the agent host. The number of items in the array determines how many dynamic ports are allocated for every task. For every port that is zero, a globally unique (cluster-wide) port is assigned and provided as part of the app definition to be used in load balancing definitions.",
      "items": {
        "maximum": 65535,
        "minimum": 0,
        "type": "integer"
      }
    },
    "portDefinitions": {
      "type": "array",
      "description": "An array of required port resources on the agent host. The number of items in the array determines how many dynamic ports are allocated for every task. For every port definition with port number zero, a globally unique (cluster-wide) service port is assigned and provided as part of the app definition to be used in load balancing definitions.",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "properties": {
          "port": {
            "type": "integer",
            "description": "if requirePorts is set to true, then this port number will be used on the agent host Otherwise if the requirePorts is set to false and this port number is not zero, then it will be used as a service port and a dynamic port will be used on the agent host. If it is set to zero, a dynamic port will be used on the host and a unique service port will be assigned by Marathon.",
            "maximum": 65535,
            "minimum": 0
          },
          "labels": {
            "type": "object",
            "description": "This can be used to add metadata to be interpreted by external applications such as firewalls.",
            "additionalProperties": {
              "type": "string"
            }
          },
          "name": {
            "type": "string",
            "description": "Name of the service hosted on this port. If provided, it must be unique over all port definitions.",
            "pattern": "^[a-z][a-z0-9-]*$"
          },
          "protocol": {
            "type": "string",
            "enum": ["tcp", "udp"],
            "description": "Protocol of the port (one of [&#39;tcp&#39;, &#39;udp&#39;]). Defaults to &#39;tcp&#39;."
          }
        }
      }
    },
    "readinessChecks": {
      "items": {
        "type": "object",
        "additionalProperties": false,
        "description": "Query these readiness checks to determine if a task is ready to serve requests.",
        "properties": {
          "name": {
            "type": "string",
            "description": "The name used to identify this readiness check."
          },
          "protocol": {
            "type": "string",
            "description": "Protocol of the requests to be performed. One of HTTP, HTTPS.",
            "enum": ["HTTP", "HTTPS"]
          },
          "path": {
            "type": "string",
            "description": "Path to endpoint exposed by the task that will provide readiness status. Example: /path/to/health."
          },
          "portName": {
            "type": "string",
            "description": "Name of the port to query as described in the portDefinitions. Example: http-api."
          },
          "intervalSeconds": {
            "type": "integer",
            "description": "Number of seconds to wait between readiness checks. Defaults to 30 seconds",
            "minimum": 0
          },
          "timeoutSeconds": {
            "type": "integer",
            "description": "Number of seconds after which a health check is considered a failure regardless of the response. Must be smaller than intervalSeconds. Defaults to 10 seconds.",
            "minimum": 0
          },
          "httpStatusCodesForReady": {
            "items": {
              "type": "integer",
              "minimum": 100,
              "maximum": 999
            },
            "description": "The HTTP(s) status code to treat as &#39;ready&#39;.",
            "type": "array"
          },
          "preserveLastResponse": {
            "type": "boolean",
            "description": "If and only if true, preserve the last readiness check responses and expose them in the API as part of a deployment.."
          }
        }
      },
      "type": "array"
    },
    "residency": {
      "type": "object",
      "description": "When using local persistent volumes that pin tasks onto agents, these values define how Marathon handles terminal states of these tasks.",
      "properties": {
        "relaunchEscalationTimeoutSeconds": {
          "type": "integer",
          "description": "When a task using persistent local volumes cannot be restarted on the agent it&#39;s been pinned to, Marathon will try to launch this task on another node after this timeout. Defaults to 3600 (one hour).",
          "minimum": 0,
          "additionalProperties": false
        },
        "taskLostBehavior": {
          "type": "string",
          "description": "When Marathon receives a TASK_LOST status update indicating that the agent running the task is gone, this property defines whether Marathon will launch the task on another node or not. Defaults to WAIT_FOREVER",
          "enum": ["WAIT_FOREVER", "RELAUNCH_AFTER_TIMEOUT"],
          "additionalProperties": false
        }
      },
      "additionalProperties": false
    },
    "requirePorts": {
      "type": "boolean",
      "description": "Normally, the host ports of your tasks are automatically assigned. This corresponds to the requirePorts value false which is the default. If you need more control and want to specify your host ports in advance, you can set requirePorts to true. This way the ports you have specified are used as host ports. That also means that Marathon can schedule the associated tasks only on hosts that have the specified ports available."
    },
    "storeUrls": {
      "type": "array",
      "description": "URL&#39;s that have to be resolved and put into the artifact store, before the task will be started.",
      "items": {
        "type": "string",
        "minLength": 1
      }
    },
    "upgradeStrategy": {
      "type": "object",
      "description": "During an upgrade all instances of an application get replaced by a new version. The upgradeStrategy controls how Marathon stops old versions and launches new versions.",
      "additionalProperties": false,
      "properties": {
        "maximumOverCapacity": {
          "type": "number",
          "description": "A number between 0 and 1 which is multiplied with the instance count. This is the maximum number of additional instances launched at any point of time during the upgrade process.",
          "maximum": 1.0,
          "minimum": 0.0
        },
        "minimumHealthCapacity": {
          "type": "number",
          "description": "A number between 0 and 1 that is multiplied with the instance count. This is the minimum number of healthy nodes that do not sacrifice overall application purpose. Marathon will make sure, during the upgrade process, that at any point of time this number of healthy instances are up.",
          "maximum": 1.0,
          "minimum": 0.0
        }
      }
    },
    "uris": {
      "type": "array",
      "description": "URIs defined here are resolved, before the application gets started. If the application has external dependencies, they should be defined here.",
      "items": {
        "type": "string",
        "minLength": 1
      }
    },
    "user": {
      "type": "string",
      "description": "The user to use to run the tasks on the agent."
    },
    "version": {
      "type": "string",
      "description": "The version of this definition.",
      "format": "date-time"
    },
    "versionInfo": {
      "type": "object",
      "description": "Detailed version information.",
      "additionalProperties": false,
      "properties": {
        "lastScalingAt": {
          "type": "string",
          "description": "Contains the time stamp of the last change to this app which was not simply a scaling or a restarting configuration.",
          "format": "date-time"
        },
        "lastConfigChangeAt": {
          "type": "string",
          "description": "Contains the time stamp of the last change including changes like scaling or restarting the app. Since our versions are time based, this is currently equal to version.",
          "format": "date-time"
        }
      }
    }
  },
  "required": [
    "id"
  ],
  "type": "object"
}
`
