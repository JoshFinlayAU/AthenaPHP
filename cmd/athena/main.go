// Command athena encodes PHP projects into Athena containers for use with the
// companion `athena` PHP decoder extension.
package main

import (
	"flag"
	"fmt"
	"os"

	"athena/internal/crypto"
	"athena/internal/encoder"
	"athena/internal/format"
	"athena/internal/keystore"
	"athena/internal/walker"
)

const usage = `athena — PHP source encoder

Usage:
  athena keygen [-key athena.key] [-header ext/athena/athena_key.h]
  athena header -key athena.key -out ext/athena/athena_key.h
  athena encode -key athena.key [-out DIR] [-skip a,b] SRC
  athena info  PATH

Commands:
  keygen   Generate a project key. Writes a raw key file (for encoding) and,
           with -header, an obfuscated C header to compile into the extension.
  header   (Re)generate the obfuscated C header from an existing key file.
  encode   Encode all .php files under SRC. In place by default, or into -out.
  info     Report whether PATH is an Athena-encoded file and its header fields.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = cmdKeygen(os.Args[2:])
	case "header":
		err = cmdHeader(os.Args[2:])
	case "encode":
		err = cmdEncode(os.Args[2:])
	case "info":
		err = cmdInfo(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n%s", os.Args[1], usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func cmdKeygen(args []string) error {
	fs := flag.NewFlagSet("keygen", flag.ExitOnError)
	keyPath := fs.String("key", "athena.key", "output raw key file")
	headerPath := fs.String("header", "", "also write obfuscated C header for the extension")
	force := fs.Bool("force", false, "overwrite existing key file")
	fs.Parse(args)

	if _, err := os.Stat(*keyPath); err == nil && !*force {
		return fmt.Errorf("%s already exists (use -force to overwrite)", *keyPath)
	}
	key, err := crypto.NewKey()
	if err != nil {
		return err
	}
	if err := keystore.Save(*keyPath, key); err != nil {
		return err
	}
	fmt.Printf("wrote key      %s (keyid %08x)\n", *keyPath, format.KeyID(key))
	if *headerPath != "" {
		if err := keystore.WriteCHeader(*headerPath, key); err != nil {
			return err
		}
		fmt.Printf("wrote C header %s\n", *headerPath)
	}
	return nil
}

func cmdHeader(args []string) error {
	fs := flag.NewFlagSet("header", flag.ExitOnError)
	keyPath := fs.String("key", "athena.key", "key file from `athena keygen`")
	out := fs.String("out", "ext/athena/athena_key.h", "output C header path")
	fs.Parse(args)

	key, err := keystore.Load(*keyPath)
	if err != nil {
		return err
	}
	if err := keystore.WriteCHeader(*out, key); err != nil {
		return err
	}
	fmt.Printf("wrote C header %s (keyid %08x)\n", *out, format.KeyID(key))
	return nil
}

func cmdEncode(args []string) error {
	fs := flag.NewFlagSet("encode", flag.ExitOnError)
	keyPath := fs.String("key", "athena.key", "key file from `athena keygen`")
	out := fs.String("out", "", "output directory (default: encode in place)")
	skip := fs.String("skip", "", "comma-separated extra paths to exclude")
	quiet := fs.Bool("quiet", false, "suppress per-file output")
	fs.Parse(args)

	if fs.NArg() != 1 {
		return fmt.Errorf("encode requires exactly one SRC path")
	}
	src := fs.Arg(0)
	key, err := keystore.Load(*keyPath)
	if err != nil {
		return err
	}
	opt := walker.DefaultOptions()
	if *skip != "" {
		opt.Extra = splitCSV(*skip)
	}
	var logf func(string)
	if !*quiet {
		logf = func(s string) { fmt.Println(s) }
	}
	st, err := encoder.EncodeProject(src, *out, key, opt, logf)
	if err != nil {
		return err
	}
	fmt.Printf("done: %d encoded, %d skipped, %d bytes written\n", st.Encoded, st.Skipped, st.Bytes)
	return nil
}

func cmdInfo(args []string) error {
	fs := flag.NewFlagSet("info", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() != 1 {
		return fmt.Errorf("info requires exactly one PATH")
	}
	data, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return err
	}
	idx := indexOf(data, format.Magic)
	if idx < 0 {
		fmt.Println("not an Athena-encoded file")
		return nil
	}
	h, err := format.ParseHeader(data[idx:])
	if err != nil {
		return err
	}
	fmt.Printf("Athena-encoded file\n")
	fmt.Printf("  container offset %d\n", idx)
	fmt.Printf("  version          %d\n", h.Version)
	fmt.Printf("  flags            0x%02x\n", h.Flags)
	fmt.Printf("  keyid            %08x\n", h.KeyID)
	fmt.Printf("  orig length      %d bytes\n", h.OrigLen)
	return nil
}

func splitCSV(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ',' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func indexOf(hay, needle []byte) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		ok := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}
