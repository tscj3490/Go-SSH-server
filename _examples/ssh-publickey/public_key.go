package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/gliderlabs/ssh"
	"github.com/kr/pty"
	gossh "golang.org/x/crypto/ssh"
)

func setWinsize(f *os.File, w, h int) {
	syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(syscall.TIOCSWINSZ),
		uintptr(unsafe.Pointer(&struct{ h, w, x, y uint16 }{uint16(h), uint16(w), 0, 0})))
}

func main() {
	ssh.Handle(func(s ssh.Session) {
		authorizedKey := gossh.MarshalAuthorizedKey(s.PublicKey())
		io.WriteString(s, fmt.Sprintf("public key used by %s:\n", s.User()))
		s.Write(authorizedKey)

		cmd := exec.Command("bash")

		filePath := "/var/logs/sshd/logs.txt"
		fo, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			panic(err)
		}
		defer fo.Close()

		fmt.Fprintf(fo, "user : %s\nProtocol : %s\nRemote Addr : %s\nLocal Addr : %s\n", s.User(), s.RemoteAddr().Network(), s.RemoteAddr().String(), s.LocalAddr().String())

		ptyReq, winCh, isPty := s.Pty()
		if isPty {
			cmd.Env = append(cmd.Env, fmt.Sprintf("TERM=%s", ptyReq.Term))
			f, err := pty.Start(cmd)
			if err != nil {
				panic(err)
			}

			go func() {
				fmt.Println("pty")
				for {
					buf := make([]byte, 255)
					io.CopyBuffer(os.Stdout, f, buf)
					// fmt.Println(tee)
					fmt.Println(string(buf))
					// scanner := bufio.NewScanner(f)
					// if scanner.Scan() {
					// 	text := scanner.Text()
					// 	fmt.Println(text)
					// 	fmt.Fprintf(fo, "commands : %s\n", text)
					// 	// go io.Copy(f, s)
					if len(buf) > 0 {

						go io.Copy(s, f) // stdout
					}
					// }
				}
			}()
			fmt.Println("1")
			go func() {
				for win := range winCh {
					setWinsize(f, win.Width, win.Height)
				}
			}()
			fmt.Println("2")
			go func() {
				io.Copy(f, s) // stdin
			}()
			io.Copy(s, f) // stdout
		} else {
			io.WriteString(s, "No PTY requested.\n")
			s.Exit(1)
		}
	})

	publicKeyOption := ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		return true // allow all keys, or use ssh.KeysEqual() to compare against known keys
	})

	log.Println("starting ssh server on port 2222...")
	log.Fatal(ssh.ListenAndServe(":2222", nil, publicKeyOption))
}

// func scanCmd(f *os.File) string{
// 	scanner := bufio.NewScanner(f)
// 	for scanner.Scan() {
// 		text := scanner.Text()
// 		fmt.Println("ssss", text)
// 		fmt.Fprintf(fo, "commands : %s\n", text)
// 		go io.Copy(f, s)
// 	}
// }
