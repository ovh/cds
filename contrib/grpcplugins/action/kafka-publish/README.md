# Plugin Kafka Publish

This CDS Action Plugin helps you to publish data through Apache Kafka in a CDS Pipeline.

## How to build

Make sure go >=1.10 is installed and properly configured ($GOPATH must be set)

```shell
    $ mkdir -p $GOPATH/src/github.com/ovh/cds
    $ git clone https://github.com/ovh/cds $GOPATH/src/github.com/ovh/cds
    $ cd $GOPATH/src/github.com/ovh/cds/contrib/grpcplugins/action
    $ make build kafka-publish
```

## How to install

### Install the binary file

```shell
    $ cd $GOPATH/src/github.com/ovh/cds/contrib/grpcplugins/action/kafka-publish
    $ make publish kafka-publish
```

## How to use

### Consumer Side

Communication between consumer and producer can be encrypted with GPG or AES.

- GPG: use flags `--pgp-decrypt`
- AES: use flags `--key`

#### Generate a new GPG Key

In a terminal

```shell
    $ gpg --gen-key
```

At the prompt, specify the kind of key you want, or press `Enter` to accept the default.

Enter the desired key size. We recommend the maximum key size of `4096`.

Press Enter to specify the default selection, indicating that the key doesn't expire.

Verify that your selections are correct.

Enter your user ID information.

Type a secure passphrase.

```shell
    gpg --list-secret-keys
```

This will shows all gpg keys. Please note the key ID you have just created.

#### Export your public key in ASCII armored format

In a terminal

```shell
    $ gpg --export --armor <KEY_ID> > ~/gpg.pub.asc
```

#### Export your private key in ASCII armored format

In a terminal

```shell
    $ gpg --export-secret-key --armor <KEY_ID> > ~/gpg.priv.asc
```

#### Listen for incoming CDS data

In a terminal, go to the working directory in which you want to receive all CDS Data and run:

```shell
    $ plugin-kafka-publish listen <kafka address> <topic> <group> <kafka user> --kafka-password <kafka password> --pgp-decrypt ~/gpg.priv.as
```

Enter your secure passphrase. You now should be able to see it action...

```shell
    $ plugin-kafka-publish listen kafka.queue.ovh.net:9000 myapp.my-topic myapp.my-topic.cds.group myapp.my-topic.cds my-user --kafka-password  ************************ --pgp-decrypt ~/gpg.priv.asc
    Please enter your passphrase: ************
    Listening Kafka kafka.queue.ovh.net:9000 on topic myapp.my-topic...
```

Now the listener will listen for data send by CDS. Data send by CDS are composed of :

- Context
- Files

Prior to files the listener should receive a context from CDS. This context will be printed on your terminal :

```shell
    $ plugin-kafka-publish listen kafka.queue.ovh.net:9000 myapp.my-topic myapp.my-topic.cds.group myapp.my-topic.cds my-user --kafka-password ************************ --pgp-decrypt ~/gpg.priv.asc
    Please enter your passphrase: ************
    Listening Kafka kafka.queue.ovh.net:9000 on topic myapp.my-topic...
    New Context received : {"action_id":1220,"directory":"1220","files":["message","file_1"]}
```

After that, the listener should receive files. Every file should be printed in your terminal :

```shell
    $ plugin-kafka-publish listen kafka.queue.ovh.net:9000 myapp.my-topic myapp.my-topic.cds.group myapp.my-topic.cds my-user --kafka-password ************************ --pgp-decrypt ~/gpg.priv.asc
    Please enter your passphrase: ************
    Listening Kafka kafka.queue.ovh.net:9000 on topic myapp.my-topic...
    New Context received : {"action_id":1220,"directory":"1220","files":["message","file_1"]}
    Received file message in context 1220 (1220/message)
    Received file fichier in context 1220 (1220/file_1)
    Context 1220 successfully closed
```

Note that the context is mark as **closed**. It means that all file have been received and are available.
In the current directory. a new file and a new directory have been created :

```shell
    $ ls
    cds-action-1220.json
    1220
    $ cat cds-action-1220.json
    {"action_id":1220,"directory":"1220","files":["message","file_1"]}
    $ ls 1220/
    message
    file_1
```

The JSON file is the CDS context. It says that the context is related to `action_id=1220` is CDS Engine, and files are stored in `1220` directory. Files are `message` and `file_1`.
From the consumer side, if you need to trigger a piece of script, you should just have to watch for new incoming json file.

The listener will never delete files, so have to do it by yourself.

#### Send acknowledgement to CDS

If you want to send acknowledgement to the CDS action which triggered the files transfert you can do it with :

```shell
    $ plugin-kafka-publish ack kafka.queue.ovh.net:9000 myapp.my-topic-ack my-user ./cds-action-1220.json OK --log my_log_file --kafka-password ************************ --artifact file1  --artifact file2  --artifact file3
```

You have to specify which CDS context you want to ack, using the previously created file (`cds-action-1220.json`), then the status of the action `OK` or `KO`. You can also attach a log file : it will be accessible in logs from CDS; and you can upload to CDS as many artifact as you want.

#### Exec on complete receiving

If you want to run a script or a binary at the end of each CDS tranfert you can do it with :

```shell
    $ plugin-kafka-publish listen kafka.queue.ovh.net:9000 myapp.my-topic myapp.my-topic.cds.group myapp.my-topic.cds my-user --kafka-password ************************ --pgp-decrypt ~/gpg.priv.asc --exec ./myScript.sh
    Please enter your passphrase: ************
    Listening Kafka kafka.queue.ovh.net:9000 on topic myapp.my-topic...
    New Context received : {"action_id":1220,"directory":"1220","files":["message","file_1"]}
    Received file message in context 1220 (1220/message)
    Received file fichier in context 1220 (1220/file_1)
    Context 1220 successfully closed
```

with, for instance `myScript.sh` as following :

```shell
    #!/bin/bash
    echo "I am the script"
    echo $0
    echo $1
    cat $1
```

It run this script with the context file as first argument. So you can get the json file, parse it and do what you want with all the files and artifacts.

Our example will prompt:

```shell
    $ plugin-kafka-publish listen kafka.queue.ovh.net:9000 myapp.my-topic myapp.my-topic.cds.group myapp.my-topic.cds my-user --kafka-password ************************ --pgp-decrypt ~/gpg.priv.asc --exec ./myScript.sh
    Please enter your passphrase: ************
    Listening Kafka kafka.queue.ovh.net:9000 on topic myapp.my-topic...
    New Context received : {"action_id":1220,"directory":"1220","files":["message","file_1"]}
    Received file message in context 1220 (1220/message)
    Received file fichier in context 1220 (1220/file_1)
    Context 1220 successfully closed
    I am the script
    ./myScript.sh
    cds-action-1220.json
    {"action_id":1220,"directory":"1220","files":["message","file_1"]}
```

### Producer Side

In a CDS Pipeline Job add a `plugin-kafka-publish` action and set the following parameters :

- `kafkaAddresses` : Set the Kafka address (ex : `kafka.queue.ovh.net:9000`)
- `topic` : Set the Kafka topic in which CDS will send the files (ex: `myapp.my-topic`)
- `kafkaUser` : Set the user to connect to kafka. Please use a CDS variable.
- `kafkaPassword` : Set the password to connect to kafka. Please use a CDS variable.
- `kafkaGroup` : Kafka Consumer Group (used for acknowledgment)
- `message` : The `message` file, you can template it and use CDS variables. Default is json format, but you can set every thing you want.
- `artifacts` : Set the list of files you want to send. In the example abose, the list should be just `file_1` because file `message` is always sent. If your want to send artifacts built elsewhere in you pipeline, don't forget to add Download Artifact action prior to this one. The list is comma separated.
- `publicKey` : Set the CDS variable is which you store you GPG public key (ex: `{{.cds.prog.gpgkey}}`). Set the value of the key to the content of the `gpg.pub.asc` file previously created.
- `waitForAck` : Set if you want to wait for an ack from the consumer side.
- `waitForAckTopic` : The kafka topic in which you will send acks. It can't be the same as the `{{.topic}}`
- `waitForAckTimeout`: If ack is received after the timeout, CDS wil consider pipeline as failed.


### Alternative ways of passing kafka password

Kafka password can be passed directly from the command line. In case this poses a security risk, it can also be read from `CDS_KAFKA_PASSWORD` environment variable.