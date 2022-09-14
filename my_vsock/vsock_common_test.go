package my_vsock

import (
	"crypto/md5"
	"crypto/sha256"
	"log"
	"testing"
)

func TestStringsHash(t *testing.T) {
	s1 := "nihadfha;lskdjf;alsdjf;asdk"
	s2 := "你是个大坏蛋hhhhhhh啊是东方家啊是东方家卡上岛咖啡"
	h256S1 := sha256.Sum256([]byte(s1))
	h256S1_1 := sha256.Sum256([]byte(s1))
	var h256S2 = sha256.Sum256([]byte(s2))
	md5S1 := md5.New().Sum([]byte(s1))
	md5S2 := md5.New().Sum([]byte(s2))
	log.Printf("S1 h256 len: %d", len(h256S1))
	log.Printf("S2 h256 len: %d", len(h256S2))
	log.Printf("S1 md5 len: %d", len(md5S1))
	log.Printf("S2 md5 len: %d", len(md5S2))
	log.Printf("Two sha hash eq: %v", h256S1 == h256S1_1)
}

func TestPackageData2(t *testing.T) {
	s := "我是niiikashjdfakaskdjf;lakjdf;alksdfjajkdfajdf看见啊东方家啊；大家发来的金科分就东方；阿金科东方；阿苏就东方家；阿金科东方；阿的金科发；的金科分；阿的江风家啊；东方家卡；的金科发；就东方；阿就东方；阿的江风家啊东方哈看东方哈金科厉害东方拉宽带好发的；加法；剪短发；到家啊家；副科级啊大家发；大家发； ；阿就东方；阿就东方；阿就到；放假啊；东方家啊；大家发；阿就东方；阿就发；阿就发；阿就发酒疯；阿真是阿就发；阿就多发几东方；阿个；阿黄；发挥；发挥；俺发酒疯；阿李静发；阿就发放假啊；加法；阿就发；阿就东方；家啊东方家啊；大家发；加法；阿里；阿就说东方；阿李静发；阿就发酒疯；垃圾；放假啊加法剪短发啊加法；啊加法；啊；剪短发；挨饿去剖人清河发挥减肥啦要噶阿就师傅皮球儿童回去普通话哦去曝外人屁啊活泼热情派人去i人前仆人"
	t.Logf("s len: %d", len([]byte(s)))
	if buf, err := PackageData([]byte(s)); err != nil {
		t.Errorf("Package wrong. %v", err)
	} else {
		t.Logf("Package data len: %d", len(buf))
		if _d, err := UnPackageData(buf); err != nil {
			t.Errorf("Unpackage fialed %v", err)
		} else {
			if len(_d) != len([]byte(s)) {
				t.Errorf("len not eq %d %d", len(_d), len([]byte(s)))
			}
			t.Logf("Result is: \n %s", string(_d))
			for i, _d := range _d {
				if []byte(s)[i] != _d {
					t.Errorf("Data wrong at: %d %x:%x", i, []byte(s)[i], _d)
				}
			}
		}

	}
}
