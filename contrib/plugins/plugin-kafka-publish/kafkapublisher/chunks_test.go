package kafkapublisher

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	f := &File{
		ID:      "1",
		Name:    "test",
		Content: bytes.NewBuffer([]byte("This is a secret message.")),
	}

	body, _ := ioutil.ReadAll(f)
	assert.Equal(t, "This is a secret message.", string(body))
}

func TestChunks(t *testing.T) {
	dir, _ := os.Getwd()
	f, err := OpenFile(path.Join(dir, "types.go"))
	assert.NoError(t, err)
	assert.NotNil(t, f)

	chunks, err := f.Chunks(100)

	var size int
	var content string
	for _, c := range chunks {
		size += len(c.Content)
		content += string(c.Content)
	}
	assert.Equal(t, len(f.Content.Bytes()), size)
	assert.Equal(t, string(f.Content.Bytes()), content)

	assert.NoError(t, err)
	t.Logf("Number of chunks : %d", len(chunks))
}

func TestEncrypt(t *testing.T) {
	f := File{
		ID:      "1",
		Name:    "test",
		Content: bytes.NewBuffer([]byte("This is a secret message")),
	}

	t.Logf("==>%s", f.Content.Bytes())

	f.EncryptContent([]byte(publicKey))

	t.Logf("==>%s", f.Content.Bytes())
}

func TestEncryptContentAndGetChunks(t *testing.T) {
	dir, _ := os.Getwd()
	f, err := OpenFile(path.Join(dir, "types.go"))
	assert.NoError(t, err)
	assert.NotNil(t, f)

	chunks, err := f.Chunks(100)

	assert.NoError(t, err)
	t.Logf("Number of chunks : %d", len(chunks))

	var content string
	for _, c := range chunks {
		content += string(c.Content)
	}

	err = f.EncryptContent([]byte(publicKey))
	assert.NoError(t, err)
	chunks, err = f.Chunks(100)
	assert.NoError(t, err)
	t.Logf("Number of encrypted chunks : %d", len(chunks))

	assert.NotEqual(t, f.Content, content)
}

func TestDecryptContent(t *testing.T) {
	dir, _ := os.Getwd()
	f, err := OpenFile(path.Join(dir, "types.go"))
	assert.NoError(t, err)
	assert.NotNil(t, f)

	clearContent := f.Content.Bytes()

	err = f.EncryptContent([]byte(publicKey))
	assert.NoError(t, err)

	assert.NotEqual(t, clearContent, f.Content)

	err = f.DecryptContent([]byte(privateKey), []byte("password"))
	assert.NoError(t, err)

	assert.EqualValues(t, clearContent, f.Content.Bytes())
}

func TestChunksToFile(t *testing.T) {
	dir, _ := os.Getwd()
	f, err := OpenFile(path.Join(dir, "types.go"))
	assert.NoError(t, err)
	assert.NotNil(t, f)

	err = f.EncryptContent([]byte(publicKey))
	assert.NoError(t, err)
	var chunks Chunks
	chunks, err = f.Chunks(100)
	assert.NoError(t, err)

	f1 := &File{
		ID:           f.ID,
		Name:         f.Name,
		ChunksNumber: f.ChunksNumber,
	}
	err = chunks.Reassemble(f1)
	assert.NoError(t, err)

	assert.EqualValues(t, f.Content.Bytes(), f1.Content.Bytes())
	assert.Equal(t, len(f.Content.Bytes()), len(f1.Content.Bytes()))

}

func TestEncryptThenChunksThenDecrypt(t *testing.T) {
	dir, _ := os.Getwd()
	f, err := OpenFile(path.Join(dir, "types.go"))
	assert.NoError(t, err)
	assert.NotNil(t, f)

	clearContent := f.Content.Bytes()

	err = f.EncryptContent([]byte(publicKey))
	assert.NoError(t, err)
	chunks, err := f.Chunks(100)

	assert.NoError(t, err)

	f1 := &File{
		ID:           f.ID,
		Name:         f.Name,
		ChunksNumber: f.ChunksNumber,
	}
	err = Chunks(chunks).Reassemble(f1)
	assert.NoError(t, err)

	err = f1.DecryptContent([]byte(privateKey), []byte("password"))
	assert.NoError(t, err)

	assert.EqualValues(t, clearContent, f1.Content.Bytes())

}

//THIS IS A TEST KEY. DON'T USE IT IN PRODUCTION
const publicKey = `-----BEGIN PGP PUBLIC KEY BLOCK-----
Version: GnuPG v2.0.22 (GNU/Linux)

mQENBFf/T1oBCADTEHL7MyGbqCKHMpW5UuBhx+OdOAAKl4+SSKuiqxswUX/XSUDD
3Vj4QEweOqYk1bSySAjsY+r3ICxX893Uf6e1Y1Bn7nzMM+6sJDnXkun2cmAOguI9
ng79RE/Z6zhowH6wGlnn5hh34nvfZL8eg9JXyv9oDUi5jxyqOlToPLM8b7ndA/is
hAST6FNHT/GcKvjKxiYec4EFkm+MtXdoxzheG58iPVewbo3iehby8DY2Jf4LaB63
3XecRDmqiw99LXBOvL/Ci7vavfF/VJTQJKZppFVWuDlq6qXZnC6wsqmoQZunGkvt
eBPFvzLHvSj2EoSq7bTB4ofrXDVLJ3xaLRDDABEBAAG0OUZyYW7Dp29pcyBTYW1p
biBUZXN0IChUZXN0KSA8ZnJhbmNvaXMuc2FtaW5AY29ycC5vdmguY29tPokBOQQT
AQIAIwUCV/9PWgIbAwcLCQgHAwIBBhUIAgkKCwQWAgMBAh4BAheAAAoJEHeDmHCO
Rl9gA3MH/2q6rP8A3KL8/2g3XNyqAlcXfOTWT1u1+hZcBTTAYmzLWBu/bgHfl/nP
lv1TIDUzku3LJ6iSZUSrYuqScZRNJBgE/Ce8knzfQ0Jf8fFJTKTHEpAK9g4ZXeUN
8A9enPmHszjoqxKemfqay1zc3qCAU/Crw7M5F/Nv6vod/pwdvWBrxYrROe5Jw65F
v7BN8Jc0Md7MpmU/RY0cWHgOx27gKmpRDBQ7xmCkZbwJTHMtfZN+WjfbkY1VTQMm
pY1IzpKnJuju5soEAepNQukRnC5JYpcChq+1f1svxQtI6XKe0a7L3eAXhi7rVtiG
njNhE9i9e/l2SDiMee8fFEpF4sQZQ1K5AQ0EV/9PWgEIALSiXDiyAXlM1A/Pjb0e
T/NclypOI4Eeo5mfxKSns178hehTsb01iwRTPnzs0mAIMd51rehW5rCTZ5hOyPW5
JtCluDC35rmrQuOg5C3781jTeehwe30lspt+M1yoVHbwJVr4p+j4t1aeFN/aQddd
AAGYSL/SsIbF8nhzFTaG8G/+yeF6V4ZocSHE4xuV9Fva/V/rE+sB6Cl7xuaBv9Ov
0ZHHVgvU0wV78EYBHppKN4O1YUb8i9lZ7yq+oWsw071H6VZPoUyoxf0/h//N5+pK
Jng1JEa1HjQMkOH3IC5Q1Txu2iDFOIE4wd+VpKrv2ClBqhhnPDT6h2+R5ZkKs0hf
A8sAEQEAAYkBHwQYAQIACQUCV/9PWgIbDAAKCRB3g5hwjkZfYGK2B/41kxR4CSBh
q9LgZMviL7po5wchzz1g4Mo/pxB2gGe8/lx5Ibq+mO53HvTW2NYhNsw097364cAh
lkCPMqkanbngUaU96eVlceCNYsVbYYmhRk3uPitLe3N8Ec1Md8HA0ymlm+iu8Jj0
9hLty0+IKFovMkeOzA3EvLYht6EPEe7OD1UV6tFzPEalDzcUpF9K2slXsVhfn+TG
OmXPAdz4pcOY2L71SqKILooNlcQ3T8t8OuWsPz3hqjV0Hh+jwK0XVZV37t+6ZYM7
XNbnOilf50/s48H3/QKy+irSINujkKmLCGdeqlfjbydiwIg1OlQcy8FftqwUls5C
X6KHmkaTACLv
=hHAk
-----END PGP PUBLIC KEY BLOCK-----`

//THIS IS A TEST KEY. DON'T USE IT IN PRODUCTION
const privateKey = `-----BEGIN PGP PRIVATE KEY BLOCK-----
Version: GnuPG v2.0.22 (GNU/Linux)

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
=r/OK
-----END PGP PRIVATE KEY BLOCK-----`
