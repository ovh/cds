package kafkapublisher

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/fsamin/go-shredder"
)

//MagicNumber is the magicNumber for chunks
var MagicNumber = []byte("!!CDS!!")

func contextData(ctx *shredder.Ctx) []byte {
	r := []byte{}
	uuidSize := make([]byte, 4)
	binary.LittleEndian.PutUint32(uuidSize, uint32(len(ctx.UUID)))
	r = append(r, []byte(uuidSize)...)
	r = append(r, []byte(ctx.UUID)...)
	chunksNumber := make([]byte, 4)
	binary.LittleEndian.PutUint32(chunksNumber, uint32(ctx.ChunksNumber))
	r = append(r, []byte(chunksNumber)...)
	r = append(r, []byte(ctx.ContentType)...)
	return r
}

//KafkaMessages returns bunch of bytes
func KafkaMessages(chunks shredder.Chunks) ([][]byte, error) {
	if len(chunks) == 0 {
		return nil, nil
	}

	ctx := chunks.Context()
	headerData := contextData(ctx)

	var res = [][]byte{}
	for _, c := range chunks {
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
		r := []byte{}
		r = append(r, MagicNumber...)
		//Push header size in bytes array
		r = append(r, headerSize...)
		//Push header data in bytes array
		r = append(r, headerData...)
		//Push chunck offset in bytes array
		offset := make([]byte, 4)
		binary.LittleEndian.PutUint32(offset, uint32(c.Offset))
		r = append(r, offset...)
		//Push chunck data in bytes array
		r = append(r, c.Data...)
		//Message ready
		res = append(res, r)
	}
	return res, nil
}

//ReadBytes transform a bytes array in a Chunk struct
func ReadBytes(b []byte) (*shredder.Chunk, error) {
	//Read the magic number
	magicNumber := b[:len(MagicNumber)]
	if !bytes.Equal(magicNumber, MagicNumber) {
		return nil, fmt.Errorf("Where is the magic number ?")
	}

	//Thea the headersize
	headerSizeBuff := b[len(MagicNumber) : len(MagicNumber)+4]
	var headerSize uint32
	if err := binary.Read(bytes.NewBuffer(headerSizeBuff), binary.LittleEndian, &headerSize); err != nil {
		return nil, err
	}

	//Read the header data
	headerData := b[len(MagicNumber)+4 : len(MagicNumber)+4+int(headerSize)]
	//Read the UUID size
	uuidSizeBuffer := headerData[:4]
	var uuidSize uint32
	if err := binary.Read(bytes.NewBuffer(uuidSizeBuffer), binary.LittleEndian, &uuidSize); err != nil {
		return nil, err
	}
	//Read the uuid
	uuid := headerData[4 : 4+uuidSize]
	//Read the chunksNumber
	chunksNumberBuff := headerData[4+uuidSize : 4+4+uuidSize]
	var chunksNumber uint32
	if err := binary.Read(bytes.NewBuffer(chunksNumberBuff), binary.LittleEndian, &chunksNumber); err != nil {
		return nil, err
	}
	//Read the ContentType
	contentType := headerData[4+4+uuidSize:]

	//Read the offset
	//Read the UUID size
	offsetBuffer := b[len(MagicNumber)+4+int(headerSize) : len(MagicNumber)+4+int(headerSize)+4]
	var offset uint32
	if err := binary.Read(bytes.NewBuffer(offsetBuffer), binary.LittleEndian, &offset); err != nil {
		return nil, err
	}

	chunkData := b[len(MagicNumber)+4+int(headerSize)+4:]

	ctx := &shredder.Ctx{
		ContentType:  string(contentType),
		UUID:         string(uuid),
		ChunksNumber: int(chunksNumber),
	}

	c := &shredder.Chunk{
		Ctx:    ctx,
		Offset: int(offset),
		Data:   chunkData,
	}

	return c, nil
}

//IsChunk test magic number is the bytes array
func IsChunk(data []byte) bool {
	magicNumber := data[:len(MagicNumber)]
	return bytes.Equal(magicNumber, MagicNumber)
}
