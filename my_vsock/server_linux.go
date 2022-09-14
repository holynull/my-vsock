package my_vsock

import (
	"encoding/binary"
	"encoding/hex"
	"log"
	"net"
	"os"
	"runtime"

	"github.com/mdlayher/vsock"
)

func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		return os.IsExist(err)
	}
	return true
}

type MyMessage struct {
	Data []byte
	Conn net.Conn
}

var RECV_MSG_CHAN chan MyMessage = make(chan MyMessage)

func handlerConnect(conn net.Conn) {
	log.Printf("Local address: %s", conn.LocalAddr().String())
	defer conn.Close()
	log.Println("Client connected.")
	SendMsg(CONNECTED_OK, conn)
	log.Println("Send connected ok")
	defer conn.Close()
	buf := make([]byte, 0)   // big buffer
	tmp := make([]byte, 256) // using small tmo buffer for demonstrating
	var msgMap map[string]map[string]MessagePackage = make(map[string]map[string]MessagePackage)
Read:
	for {
		n, err := conn.Read(tmp)
		if err != nil {
			log.Println("read error:", err)
			// conn.Close()
			break Read
		}
		buf = append(buf, tmp[:n]...)
		if len(buf) >= int(PACKAGE_MAX_SIZE) {
			log.Printf("Get a message package from client. buf size is: %d", len(buf))
			_d := [PACKAGE_MAX_SIZE]byte{}
			for i := range _d {
				_d[i] = buf[0:PACKAGE_MAX_SIZE][i]
			}
			mP := BytesToMessagePackage(_d)
			fHash := hex.EncodeToString(mP.FileHash[:])
			mBodyLen := binary.BigEndian.Uint32(mP.BodyLen)
			fsizeInP := binary.BigEndian.Uint32(mP.Length)
			var cLen int = 0
			var fsize uint32 = 0
			if msgMap[fHash] == nil {
				msgMap[fHash] = make(map[string]MessagePackage)
				fsize = binary.BigEndian.Uint32(mP.Length)
			}
			for _, v := range msgMap[fHash] {
				bLen := binary.BigEndian.Uint32(v.BodyLen)
				fsize = binary.BigEndian.Uint32(v.Length)
				if fsizeInP != fsize {
					buf = buf[PACKAGE_MAX_SIZE:]
					log.Println("Bad package: file size different to before.")
					continue Read
				}
				cLen += int(bLen)
			}
			pHash := hex.EncodeToString(mP.PackageHash[:])
			msgMap[fHash][pHash] = mP
			if cLen+int(mBodyLen) == int(fsize) { // 最后一个包
				packageArr := make([]MessagePackage, len(msgMap[fHash]))
				index := 0
				for k := range msgMap[fHash] {
					packageArr[index] = msgMap[fHash][k]
					index++
				}
				if _d, err := UnPackageData(packageArr); err != nil {
					log.Printf("Unpackage data failed. %v", err)
				} else {
					RECV_MSG_CHAN <- MyMessage{Data: _d, Conn: conn}
				}
				delete(msgMap, fHash)
			}
			buf = buf[PACKAGE_MAX_SIZE:]
		}
	}
	conn.Close()
}

func StartServer(port uint32) {
	defer func() {
		log.Println("Start Server Done")
		// Println executes normally even if there is a panic
		if err := recover(); err != nil {
			log.Printf("Server run time panic: %v", err)
		}
	}()
	l, err := vsock.ListenContextID(VMADDR_CID_ANY, port, nil)
	log.Printf("VSOCK address: %s", l.Addr().String())
	defer func() {
		l.Close()
	}()
	if err != nil {
		log.Printf("Start vsock server failed. %v", err)
	}
	for {
		log.Println("Wait for client connect.")
		if conn, err := l.Accept(); err != nil {
			log.Printf("Accept vsock server failed. %v", err)
		} else {
			log.Printf("Client connected. From %s", conn.RemoteAddr().String())
			go handlerConnect(conn)
			log.Printf("Routine size: %d", runtime.NumGoroutine())
		}
	}
}
