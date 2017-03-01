package kafkapublisher

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/maxwellhealth/go-gpg"
	"github.com/satori/go.uuid"
)

//OpenFile reads a file from filepath and returns a kafkapublisher.File
func OpenFile(filepath string) (*File, error) {
	t := &File{}
	btes, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	t.Content = bytes.NewBuffer(btes)
	t.ID = uuid.NewV4().String()
	t.Name = path.Base(filepath)
	return t, nil
}

//Read implements Reader interface
func (f *File) Read(p []byte) (int, error) {
	return f.Content.Read(p)
}

//Close implements Closer interface
func (f *File) Close() error {
	return nil
}

//EncryptContent encrypts with a public key
func (f *File) EncryptContent(publicKey []byte) error {
	buf := new(bytes.Buffer)
	if err := gpg.Encode(publicKey, f, buf); err != nil {
		return err
	}
	f.Content = buf
	return nil
}

//DecryptContent decrypts with a private key
func (f *File) DecryptContent(privateKey, passphrase []byte) error {
	buf := new(bytes.Buffer)
	if err := gpg.Decode(privateKey, passphrase, f, buf); err != nil {
		return err
	}
	f.Content = buf
	return nil
}

//Chunks returns a list a chunks
func (f *File) Chunks(size int64) ([]Chunk, error) {
	chunks := []Chunk{}

	var offset int64
	var offSetNumber int
	var content = f.Content.Bytes()
	for {
		c := Chunk{}
		data := make([]byte, size)
		var length = offset + size
		//Resize last chunk
		if offset+size > int64(len(content)) {
			length = int64(len(content))
			data = make([]byte, len(content[offset:length]))
		}
		n := copy(data, content[offset:length])
		if n == 0 {
			break
		}
		offset += int64(n)
		c.ContextID = f.ContextID
		c.FileID = f.ID
		c.Filename = f.Name
		c.Offset = offSetNumber
		c.Content = data
		chunks = append(chunks, c)
		offSetNumber++
	}
	f.ChunksNumber = len(chunks)
	return chunks, nil
}

//MagicNumber is the magicNumber for chunks
var MagicNumber = []byte("!!CDS!!")

//KafkaMessages returns bunch of bytes
func (f *File) KafkaMessages(size int64) ([][]byte, error) {
	chunks, err := f.Chunks(size)
	if err != nil {
		return nil, err
	}

	//Header data is a marshalled json of file descriptor
	headerData, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}

	var res = [][]byte{}
	for _, c := range chunks {
		r := []byte{}
		//Considering 4 bytes to store header size
		headerSize := make([]byte, 4)
		//Computing header size
		buf := new(bytes.Buffer)
		var num = uint32(len(headerData))
		err := binary.Write(buf, binary.LittleEndian, num)
		if err != nil {
			return nil, err
		}
		headerSize = buf.Bytes()
		//Push magic number
		r = append(r, MagicNumber...)
		//Push header size in bytes array
		r = append(r, headerSize...)
		//Push header data in bytes array
		r = append(r, headerData...)
		//Push chunck data in bytes array
		chunkData, err := json.Marshal(c)
		if err != nil {
			return nil, err
		}
		r = append(r, chunkData...)

		//Message ready
		res = append(res, r)
	}
	return res, nil
}

//ReadBytes transform a bytes array in File and Chunk struct
func ReadBytes(b []byte) (*File, *Chunk, error) {
	magicNumber := b[:len(MagicNumber)]
	if !bytes.Equal(magicNumber, MagicNumber) {
		return nil, nil, fmt.Errorf("Where is the magic number")
	}

	headerSize := b[len(MagicNumber) : len(MagicNumber)+4]
	buf := bytes.NewBuffer(headerSize)
	var num uint32
	if err := binary.Read(buf, binary.LittleEndian, &num); err != nil {
		return nil, nil, err
	}

	headerData := b[len(MagicNumber)+4 : len(MagicNumber)+4+int(num)]
	f := &File{}
	if err := json.Unmarshal(headerData, f); err != nil {
		return nil, nil, err
	}

	chunkData := b[len(MagicNumber)+4+int(num):]
	c := &Chunk{}
	if err := json.Unmarshal(chunkData, c); err != nil {
		return nil, nil, err
	}

	if f.ID != c.FileID || f.Name != c.Filename {
		return nil, nil, fmt.Errorf("Header Data and Chunk doesn't match")
	}

	return f, c, nil
}

//IsChunk test magic number is the bytes array
func IsChunk(data []byte) bool {
	magicNumber := data[:len(MagicNumber)]
	return bytes.Equal(magicNumber, MagicNumber)
}
