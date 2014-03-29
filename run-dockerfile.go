package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// オプション
type options struct {
	host string // リモートホスト
}

// 実行コンテキスト
type context struct {
	host   string // リモートホスト
	path   string // 処理中のファイル
	lineno uint   // 処理中の行番号

	env     map[string]string // 環境変数
	user    string            // ユーザ
	workdir string            // 作業ディレクトリ

	remote  bool     // リモート実行
	cmd_run []string // RUN命令
	cmd_add []string // ADD命令
}

// コマンドの使い方
func usage() {
	cmd := os.Args[0]
	fmt.Fprintf(os.Stderr, "usage: %s [options] [file...]\n", filepath.Base(cmd))
	flag.PrintDefaults()
	os.Exit(0)
}

// コマンドの実行
func (ctx *context) execl(name string, arg ...string) (*exec.Cmd, error) {
	cmd := exec.Command(name, arg...)
	out, err := cmd.Output()
	if 0 < len(out) {
		fmt.Printf("%s", out)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s:%d: %s : %s\n", ctx.path, ctx.lineno, err, arg)
	}

	return cmd, err
}

// 初期化
func (ctx *context) init() error {
	ctx.env = make(map[string]string)
	ctx.remote = (0 < len(ctx.host))

	if ctx.remote {
		ssh, err := exec.LookPath("ssh")
		if err != nil {
			return err
		}
		scp, err := exec.LookPath("scp")
		if err != nil {
			return err
		}
		ctx.cmd_run = []string{ssh, ctx.host}
		ctx.cmd_add = []string{scp}
	} else {
		ctx.cmd_run = []string{"/bin/sh", "-c"}
		ctx.cmd_add = []string{"/bin/cp"}
	}

	return nil
}

// RUN命令
func (ctx *context) run(arg string) (*exec.Cmd, error) {
	if 0 < len(ctx.workdir) {
		arg = fmt.Sprintf("cd %s; %s", ctx.workdir, arg)
	}

	return ctx.execl(ctx.cmd_run[0], append(ctx.cmd_run[1:], arg)...)
}

// ADD命令
func (ctx *context) add(src string, dst string) (*exec.Cmd, error) {
	if 0 < len(ctx.workdir) && !filepath.IsAbs(dst) {
		// コピー先のパスを生成する
		dst = filepath.Join(ctx.workdir, dst)
	}

	if ctx.remote {
		// リモート・パスを生成する
		dst = fmt.Sprintf("%s:%s", ctx.host, dst)
	}

	return ctx.execl(ctx.cmd_add[0], src, dst)
}

// ENV命令
func (ctx *context) set_env(key string, val string) {
	ctx.env[key] = val
	os.Setenv(key, val)
}

// Dockerfile の実行
func runDockerfile(path string, opts *options) error {
	var file *os.File
	var err error

	if len(path) == 0 {
		file = os.Stdin
	} else {
		file, err = os.Open(path)
		if err != nil {
			// エラー処理をする
			return err
		}
		defer file.Close()
	}

	scanner := bufio.NewScanner(file)

	// 正規表現のコンパイル
	re := regexp.MustCompile("(?i)^ *(ONBUILD +)?([A-Z]+) +([^#]+)")
	re_args := regexp.MustCompile("(?i)^([^ ]+) +(.+)")

	ctx := context{host: opts.host, path: path}

	err = ctx.init()
	if err != nil {
		return err
	}

	for scanner.Scan() {
		line := scanner.Text()
		ctx.lineno += 1
		match := re.FindStringSubmatch(line)
		if match != nil {
			//onbuild := match[1]
			instruction := match[2]
			args := match[3]

			switch instruction {
			case "FROM":
				// 何もしない
			case "MAINTAINER":
				// 何もしない
			case "RUN":
				// スクリプトを実行
				_, err := ctx.run(args)
				if err != nil {
					return err
				}
			case "CMD":
			case "EXPOSE":
				// iptables があれば書換える
			case "ENV":
				// 環境変数を設定
				match_args := re_args.FindStringSubmatch(args)
				if match_args != nil {
					ctx.set_env(match_args[1], match_args[2])
				}
			case "ADD":
				// ファイルを追加
				match_args := re_args.FindStringSubmatch(args)
				if match_args != nil {
					_, err := ctx.add(match_args[1], match_args[2])
					if err != nil {
						return err
					}
				}
			case "ENTRYPOINT":
			case "VOLUME":
				// 何もしない
			case "USER":
				// ユーザを設定
				ctx.user = args
			case "WORKDIR":
				// 作業ディレクトリを変更
				ctx.workdir = args
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	return err
}

func main() {
	var opts options

	flag.StringVar(&opts.host, "H", "", "host")
	help := flag.Bool("h", false, "help")
	flag.Parse()

	if *help {
		usage()
	}

	if len(flag.Args()) == 0 {
		runDockerfile("Dockerfile", &opts)
	} else {
		for _, v := range flag.Args() {
			runDockerfile(v, &opts)
		}
	}
}
