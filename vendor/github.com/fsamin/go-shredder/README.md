# Go-Shredder

Go-Shredder is a package which helps you to split a file, or a byte array into chunks. You wan also encrypt the content with GnuPG or AES encryption.

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](http://godoc.org/github.com/fsamin/go-shredder) [![Build Status](https://travis-ci.org/fsamin/go-shredder.svg?branch=master)](https://travis-ci.org/fsamin/go-shredder) [![Go Report Card](https://goreportcard.com/badge/github.com/fsamin/go-shredder)](https://goreportcard.com/report/github.com/fsamin/go-shredder)

## Sample usages

Shred a file, reassemble the chunks and print the file content.

```golang
    chunks, err := ShredFile("main.go", &Opts{
        ChunkSize: 100,
    })
    if err != nil {
        panic(err)
    }

    content, err := Reassemble(chunks, &Opts{
        ChunkSize: 100,
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(content.String())
```

## With Gnu-PG encryption

Shred a file with GPG encryption. You just need the public key to encrypt.

```golang
    chunks, err := ShredFile("main.go", &Opts{
        GPGEncryption: &GPGEncryption{
                PublicKey:  []byte(publicKey),
            },
        ChunkSize: 100,
    })
    if err != nil {
        panic(err)
    }
```

Reassemble the file; so you need private key and the passphrase.

```golang
    content, err := Reassemble(chunks, &Opts{
        GPGEncryption: &GPGEncryption{
            PrivateKey: []byte(privateKey),
            Passphrase: []byte("password"),
        },
        ChunkSize: 100,
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(content.String())
```

## With AES encryption


Shred a file with AES encryption.

```golang
    chunks, err := ShredFile("main.go", &Opts{
        AESEncryption:  &AESEncryption{
            key: []byte("a very very very very secret key"), //32 bytes long key
        },
        ChunkSize: 100,
    })
    if err != nil {
        panic(err)
    }
```

Reassemble the file

```golang
    content, err := Reassemble(chunks, &Opts{
        AESEncryption:  &AESEncryption{
            key: []byte("a very very very very secret key"), //32 bytes long key
        },
        ChunkSize: 100,
    })
    if err != nil {
        panic(err)
    }
    fmt.Println(content.String())
```