This library duplicates the basic behavior of the GPG library so you can use its functionality without deferring to an external executable or using a bridge to C. Basically this just wraps the lower-level functions from golang.org/x/crypto/openpgp to make it easier to use without having to write boilerplate code every time.

The functions here account for 100% of our use case for PGP-encryption, but if you have a case that these do not cover, feel free to submit a PR.

# Encoding
Use a new `&gpg.Encoder{}` or just `gpg.Encode(key []byte, src io.Reader, dest io.Writer)`. The key is the public key including headers.

The `Encode` function performs the same way as `gpg --output {out} --encrypt {file} -r {recipient/public key}`

## Example

```go
package main

import (
	"github.com/maxwellhealth/go-gpg"
	"os"
	"log"
)

func main() {
	var publicKey []byte

	// Retrieve the public key from somewhere...

	toEncrypt, err := os.OpenFile("path/to/secret/file", os.O_RDONLY, 0660)
	if err != nil {
		log.Fatal(err)
	}

	destination, err := os.OpenFile("path/to/destination", os.O_WRONLY, 0660)
	if err != nil {
		log.Fatal(err)
	}

	// Encrypt...
	err = gpg.Encode(publicKey, toEncrypt, destination)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Encrypted file!")
}
```


# Decoding
Use a new `&gpg.Decoder{}` or just `gog.Decode(key, passphrase []byte, src io.Reader, dest io.Writer)`. The key is the private key including headers. If the private key is encrypted with a passphrase you should provide it here, otherwise pass an empty byte slice (`[]byte{}`) as the second argument.

The `Decode` function performs the same way as `gpg --output {out} --passphrase {passphrase} --decrypt {file}`, but requires that you provide the private key rather than have it in the keyring.

## Example
```go
package main

import (
	"github.com/maxwellhealth/go-gpg"
	"os"
	"log"
)

func main() {
	var privateKey []byte
	passphrase := []byte("timbuktu")
	// Retrieve the private key from somewhere...

	toDecrypt, err := os.OpenFile("path/to/encrypted/file", os.O_RDONLY, 0660)
	if err != nil {
		log.Fatal(err)
	}

	destination, err := os.OpenFile("path/to/destination", os.O_WRONLY, 0660)
	if err != nil {
		log.Fatal(err)
	}

	// Decrypt...
	err = gpg.Decode(privateKey, passphrase, toDecrypt, destination)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Decrypted file!")
}
```