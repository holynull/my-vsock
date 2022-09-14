package my_vsock

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"

	"github.com/spf13/viper"
)

const CONNECTED_OK = "CONNECT_OK"
const VSOCK_EOF = "EOF"
const MAX_BYTE_LENGTH_OF_DATA = 4294967295
const VMADDR_CID_ANY uint32 = 4294967295
const VMADDR_CID_HYPERVISOR uint32 = 0
const VMADDR_CID_HOST uint32 = 2
const VMADDR_CID_RESERVED = 1
const VMADDR_PORT_ANY uint32 = 4294967295

type PackageHeader struct {
	FileHash    [32]byte
	PackageHash [32]byte
	Length      []byte // 4 bytes
	PackageId   []byte // 4 bytes
	BodyLen     []byte // 4 bytes
}

const BODY_MAX_SIZE = 256                                                                                                  // 包的body的最大size；
const FILE_HASH_LEN = 32                                                                                                   // file的hash的长度
const FILE_SIZE_LEN = 4                                                                                                    // file长度占用字节数
const PACKAGE_HASH_LEN = 32                                                                                                // 包的hash的长度
const PACKAGE_ID_LEN = 4                                                                                                   // 包id占用的字节数
const BODY_SIZE_LEN = 4                                                                                                    // body实际长度占用的字节数
const PACKAGE_MAX_SIZE = FILE_HASH_LEN + FILE_SIZE_LEN + PACKAGE_HASH_LEN + PACKAGE_ID_LEN + BODY_SIZE_LEN + BODY_MAX_SIZE // 包的长度

type MessagePackage struct {
	*PackageHeader
	Body [BODY_MAX_SIZE]byte
}

func (m MessagePackage) toBytes() []byte {
	r := append(m.FileHash[:], m.Length...)
	r = append(r, m.PackageHash[:]...)
	r = append(r, m.PackageId[:]...)
	r = append(r, m.BodyLen[:]...)
	r = append(r, m.Body[:]...)
	return r
}

type MessagePackageSlice []MessagePackage

func (m MessagePackageSlice) Len() int { // 重写 Len() 方法
	return len(m)
}
func (m MessagePackageSlice) Swap(i, j int) { // 重写 Swap() 方法
	m[i], m[j] = m[j], m[i]
}
func (m MessagePackageSlice) Less(i, j int) bool { // 重写 Less() 方法，升序
	return binary.BigEndian.Uint32(m[i].PackageId) < binary.BigEndian.Uint32(m[j].PackageId)
}
func IntToBytes(n uint32) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, n)
	return bytesBuffer.Bytes()
}

func PackageData(data []byte) ([]MessagePackage, error) {
	if len(data) > MAX_BYTE_LENGTH_OF_DATA {
		return nil, errors.New("data length is big than MAX_BYTE_LENGTH_OF_DATA")
	} else {
		mLen := len(data)
		FileHash := sha256.Sum256(data)
		index := 0
		msgArr := make([]MessagePackage, 0)
		for cursor := 0; cursor < len(data); {
			var body []byte
			if len(data[cursor:]) < int(BODY_MAX_SIZE) {
				body = data[cursor:]
			} else {
				body = data[cursor : cursor+int(BODY_MAX_SIZE)]
			}
			PackageHash := sha256.Sum256(body)
			packageId := IntToBytes(uint32(index))
			bodyLen := len(body)
			h := PackageHeader{FileHash: FileHash, PackageHash: PackageHash, Length: IntToBytes(uint32(mLen)), PackageId: packageId, BodyLen: IntToBytes(uint32(bodyLen))}
			empty := [BODY_MAX_SIZE]byte{}
			for i := range empty {
				if i < len(body) {
					empty[i] = body[i]
				}
			}
			p := MessagePackage{
				&h,
				empty,
			}
			index++
			cursor = cursor + len(p.Body)
			msgArr = append(msgArr, p)
		}
		return msgArr, nil
	}
}

func UnPackageData(data []MessagePackage) ([]byte, error) {
	for i, m := range data {
		if i > 0 {
			if m.FileHash != data[i-1].FileHash {
				return nil, fmt.Errorf("PACKAGE_HASH_DIFFERENT:%d", i)
			}
		}
	}
	sort.Sort(MessagePackageSlice(data)) // 按Id升序排序
	buf := []byte{}
	for _, m := range data {
		bLen := int(binary.BigEndian.Uint32(m.BodyLen))
		buf = append(buf, m.Body[0:bLen]...)
	}
	return buf, nil
}

var Config *viper.Viper

func InitViper() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Load config failed. %v", err)
	}
	Config = viper.GetViper()
}

func SendMsg(msg string, conn net.Conn) {
	if d, err := PackageData([]byte(msg)); err != nil {
		log.Printf("Package data error: %v", err)
	} else {
		for _, _d := range d {
			if _, err := conn.Write(_d.toBytes()); err != nil {
				log.Printf("Send data to server error: %v", err)
			}
		}
	}
}

func BytesToMessagePackage(data [PACKAGE_MAX_SIZE]byte) MessagePackage {
	var FileHash [FILE_HASH_LEN]byte = [FILE_HASH_LEN]byte{}
	for i := range FileHash {
		FileHash[i] = data[0:FILE_HASH_LEN][i]
	}
	var length [FILE_SIZE_LEN]byte = [FILE_SIZE_LEN]byte{}
	for i := range length {
		length[i] = data[FILE_HASH_LEN : FILE_HASH_LEN+FILE_SIZE_LEN][i]
	}
	var PackageHash [PACKAGE_HASH_LEN]byte = [PACKAGE_HASH_LEN]byte{}
	for i := range PackageHash {
		PackageHash[i] = data[FILE_HASH_LEN+FILE_SIZE_LEN : FILE_HASH_LEN+FILE_SIZE_LEN+PACKAGE_HASH_LEN][i]
	}
	var pId [PACKAGE_ID_LEN]byte = [PACKAGE_ID_LEN]byte{}
	for i := range pId {
		pId[i] = data[FILE_HASH_LEN+FILE_SIZE_LEN+PACKAGE_HASH_LEN : FILE_HASH_LEN+FILE_SIZE_LEN+PACKAGE_HASH_LEN+PACKAGE_ID_LEN][i]
	}
	var bLen [BODY_SIZE_LEN]byte = [BODY_SIZE_LEN]byte{}
	for i := range bLen {
		bLen[i] = data[FILE_HASH_LEN+FILE_SIZE_LEN+PACKAGE_HASH_LEN+PACKAGE_ID_LEN : FILE_HASH_LEN+FILE_SIZE_LEN+PACKAGE_HASH_LEN+PACKAGE_ID_LEN+BODY_SIZE_LEN][i]
	}
	var body [BODY_MAX_SIZE]byte = [BODY_MAX_SIZE]byte{}
	for i := range body {
		body[i] = data[FILE_HASH_LEN+FILE_SIZE_LEN+PACKAGE_HASH_LEN+PACKAGE_ID_LEN+BODY_SIZE_LEN : FILE_HASH_LEN+FILE_SIZE_LEN+PACKAGE_HASH_LEN+PACKAGE_ID_LEN+BODY_SIZE_LEN+BODY_MAX_SIZE][i]
	}
	header := PackageHeader{
		FileHash:    FileHash,
		PackageHash: PackageHash,
		Length:      length[:],
		PackageId:   pId[:],
		BodyLen:     bLen[:],
	}
	return MessagePackage{
		&header,
		body,
	}
}
