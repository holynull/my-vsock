package my_vsock

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Dial creates a connection to the VM with the given client ID and port
// Only support docker container
func Dial(cid, port uint32) (net.Conn, error) {
	socketPath := filepath.Join(os.Getenv("HOME"), "/Library/Containers/com.docker.docker/Data/vms/0")
	connectPath := filepath.Join(socketPath, "connect")
	c, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: connectPath, Net: "unix"})
	if err != nil {
		return c, errors.Wrapf(err, "failed to dial on %s", connectPath)
	}
	if _, err := fmt.Fprintf(c, "%08x.%08x\n", cid, port); err != nil {
		return c, errors.Wrapf(err, "Failed to write dest (%08x.%08x) to %s", cid, port, connectPath)
	}
	return c, nil
}

var MSG_FROM_SERVER_CHAN chan MyMessage = make(chan MyMessage)

func ConnetctServer(cid, port uint32) (net.Conn, error) {
	if conn, err := Dial(cid, port); err != nil {
		log.Printf("Dial failed %v", err)
		return conn, err
	} else {
		log.Println("Dial success")
		log.Printf("Remote: %s", conn.RemoteAddr().String())
		log.Printf("Local: %s", conn.LocalAddr().String())
		go func(conn net.Conn) {
			defer conn.Close()
			buf := make([]byte, 0, MAX_BYTE_LENGTH_OF_DATA) // big buffer
			tmp := make([]byte, 256)                        // using small tmo buffer for demonstrating
			var msgMap map[string]map[string]MessagePackage = make(map[string]map[string]MessagePackage)
		ReadData:
			for {
				n, err := conn.Read(tmp)
				if err != nil {
					log.Printf("read error: %v", err)
					MSG_FROM_SERVER_CHAN <- MyMessage{Data: []byte(VSOCK_EOF), Conn: conn}
					break ReadData
				}
				buf = append(buf, tmp[:n]...)
				if len(buf) >= int(PACKAGE_MAX_SIZE) {
					log.Printf("Get a message package from server. buf size is: %d", len(buf))
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
							continue ReadData
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
							MSG_FROM_SERVER_CHAN <- MyMessage{Data: _d, Conn: conn}
						}
						delete(msgMap, fHash)
					}
					buf = buf[PACKAGE_MAX_SIZE:]
				}
			}
			conn.Close()
		}(conn)
		return conn, err
	}
}
