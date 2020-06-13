package encodehelper

import (
	"crypto/ed25519"
	"math/rand"

	"golang.org/x/crypto/ssh"
)

// MarshalED25519PrivateKey to marshal ed25519 private key to openssh format
func MarshalED25519PrivateKey(key ed25519.PrivateKey) []byte {
	pk1 := struct {
		Check1  uint32
		Check2  uint32
		Keytype string
		Pub     []byte
		Priv    []byte
		Comment string
		Pad     []byte `ssh:"rest"`
	}{}

	ci := rand.Uint32()
	pk1.Check1 = ci
	pk1.Check2 = ci
	pk1.Keytype = ssh.KeyAlgoED25519
	pk1.Pub = key.Public().(ed25519.PublicKey)
	pk1.Priv = key
	pk1.Comment = ""

	// Add some padding to match the encryption block size within PrivKeyBlock (without Pad field)
	// 8 doesn't match the documentation, but that's what ssh-keygen uses for unencrypted keys. *shrug*
	bs := 8
	blockLen := len(ssh.Marshal(pk1))
	padLen := (bs - (blockLen % bs)) % bs
	pk1.Pad = make([]byte, padLen)
	for i := 0; i < padLen; i++ {
		pk1.Pad[i] = byte(i + 1)
	}

	var w struct {
		CipherName   string
		KdfName      string
		KdfOpts      string
		NumKeys      uint32
		PubKey       []byte
		PrivKeyBlock []byte
	}

	w.CipherName = "none"
	w.KdfName = "none"
	w.KdfOpts = ""
	w.NumKeys = 1

	w.PrivKeyBlock = ssh.Marshal(pk1)
	w.PubKey = ssh.Marshal(struct {
		Keytype  string
		KeyBytes []byte
	}{
		Keytype:  ssh.KeyAlgoED25519,
		KeyBytes: pk1.Pub,
	})

	return append([]byte("openssh-key-v1\x00"), ssh.Marshal(w)...)
}
