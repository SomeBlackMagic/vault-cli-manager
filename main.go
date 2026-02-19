package main

import (
	"crypto/x509"
	"io/ioutil"
	"os"
	"strings"

	fmt "github.com/jhunt/go-ansi"
	"github.com/jhunt/go-cli"
	env "github.com/jhunt/go-envirotron"
	"github.com/starkandwayne/safe/rc"
	"github.com/starkandwayne/safe/vault"
)

var Version string

func connect(auth bool) *vault.Vault {
	var caCertPool *x509.CertPool
	if os.Getenv("VAULT_CACERT") != "" {
		contents, err := ioutil.ReadFile(os.Getenv("VAULT_CACERT"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "@R{!! Could not read CA certificates: %s}", err.Error())
		}

		caCertPool = x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(contents)
	}

	shouldSkipVerify := func() bool {
		skipVerifyVal := os.Getenv("VAULT_SKIP_VERIFY")
		if skipVerifyVal != "" && skipVerifyVal != "false" {
			return true
		}
		return false
	}

	conf := vault.VaultConfig{
		URL:        getVaultURL(),
		Token:      os.Getenv("VAULT_TOKEN"),
		Namespace:  os.Getenv("VAULT_NAMESPACE"),
		SkipVerify: shouldSkipVerify(),
		CACerts:    caCertPool,
	}

	if auth && conf.Token == "" {
		fmt.Fprintf(os.Stderr, "@R{You are not authenticated to a Vault.}\n")
		fmt.Fprintf(os.Stderr, "Try @C{safe auth ldap}\n")
		fmt.Fprintf(os.Stderr, " or @C{safe auth github}\n")
		fmt.Fprintf(os.Stderr, " or @C{safe auth okta}\n")
		fmt.Fprintf(os.Stderr, " or @C{safe auth token}\n")
		fmt.Fprintf(os.Stderr, " or @C{safe auth userpass}\n")
		fmt.Fprintf(os.Stderr, " or @C{safe auth approle}\n")
		os.Exit(1)
	}

	v, err := vault.NewVault(conf)
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!! %s}\n", err)
		os.Exit(1)
	}
	return v
}

// Exits program with error if no Vault targeted
func getVaultURL() string {
	ret := os.Getenv("VAULT_ADDR")
	if ret == "" {
		fmt.Fprintf(os.Stderr, "@R{You are not targeting a Vault.}\n")
		fmt.Fprintf(os.Stderr, "Try @C{safe target https://your-vault alias}\n")
		fmt.Fprintf(os.Stderr, " or @C{safe target alias}\n")
		os.Exit(1)
	}
	return ret
}

type Options struct {
	Insecure     bool `cli:"-k, --insecure"`
	Version      bool `cli:"-v, --version"`
	Help         bool `cli:"-h, --help"`
	Clobber      bool `cli:"--clobber, --no-clobber"`
	SkipIfExists bool
	Quiet        bool `cli:"--quiet"`

	// Behavour of -T must chain through -- separated commands.  There is code
	// that relies on this.  Will default to $SAFE_TARGET if it exists, or
	// the current safe target otherwise.
	UseTarget string `cli:"-T, --target" env:"SAFE_TARGET"`

	HelpCommand    struct{} `cli:"help"`
	VersionCommand struct{} `cli:"version"`

	Envvars struct{} `cli:"envvars"`
	Targets struct {
		JSON bool `cli:"--json"`
	} `cli:"targets"`

	Status struct {
		ErrorIfSealed bool `cli:"-e, --err-sealed"`
	} `cli:"status"`

	Unseal struct{} `cli:"unseal"`
	Seal   struct{} `cli:"seal"`
	Env    struct {
		ForBash bool `cli:"--bash"`
		ForFish bool `cli:"--fish"`
		ForJSON bool `cli:"--json"`
	} `cli:"env"`

	Auth struct {
		Path string `cli:"-p, --path"`
		JSON bool   `cli:"--json"`
	} `cli:"auth, login"`

	Logout struct{} `cli:"logout"`
	Renew  struct{} `cli:"renew"`
	Ask    struct{} `cli:"ask"`
	Set    struct{} `cli:"set, write"`
	Paste  struct{} `cli:"paste"`
	Exists struct{} `cli:"exists, check"`

	Local struct {
		As     string `cli:"--as"`
		File   string `cli:"-f, --file"`
		Memory bool   `cli:"-m, --memory"`
		Port   int    `cli:"-p, --port"`
	} `cli:"local"`

	Init struct {
		Single    bool `cli:"-s, --single"`
		NKeys     int  `cli:"--keys"`
		Threshold int  `cli:"--threshold"`
		JSON      bool `cli:"--json"`
		Sealed    bool `cli:"--sealed"`
		NoMount   bool `cli:"--no-mount"`
		Persist   bool `cli:"--persist, --no-persist"`
	} `cli:"init"`

	Rekey struct {
		NKeys     int      `cli:"--keys, --num-unseal-keys"`
		Threshold int      `cli:"--threshold, --keys-to-unseal"`
		GPG       []string `cli:"--gpg"`
		Persist   bool     `cli:"--persist, --no-persist"`
	} `cli:"rekey"`

	Get struct {
		KeysOnly bool `cli:"--keys"`
		Yaml     bool `cli:"--yaml"`
	} `cli:"get, read, cat"`

	Versions struct{} `cli:"versions,revisions"`

	List struct {
		Single bool `cli:"-1"`
		Quick  bool `cli:"-q, --quick"`
	} `cli:"ls"`

	Paths struct {
		ShowKeys bool `cli:"--keys"`
		Quick    bool `cli:"-q, --quick"`
	} `cli:"paths"`

	Tree struct {
		ShowKeys   bool `cli:"--keys"`
		HideLeaves bool `cli:"-d, --hide-leaves"`
		Quick      bool `cli:"-q, --quick"`
	} `cli:"tree"`

	Target struct {
		JSON        bool     `cli:"--json"`
		Interactive bool     `cli:"-i, --interactive"`
		Strongbox   bool     `cli:"-s, --strongbox, --no-strongbox"`
		CACerts     []string `cli:"--ca-cert"`
		Namespace   string   `cli:"-n, --namespace"`

		Delete struct{} `cli:"delete, rm"`
	} `cli:"target"`

	Delete struct {
		Recurse bool `cli:"-R, -r, --recurse"`
		Force   bool `cli:"-f, --force"`
		Destroy bool `cli:"-D, -d, --destroy"`
		All     bool `cli:"-a, --all"`
	} `cli:"delete, rm"`

	Undelete struct {
		All bool `cli:"-a, --all"`
	} `cli:"undelete, unrm, urm"`

	Revert struct {
		Deleted bool `cli:"-d, --deleted"`
	} `cli:"revert"`

	Export struct {
		All     bool `cli:"-a, --all"`
		Deleted bool `cli:"-d, --deleted"`
		//These do nothing but are kept for backwards-compat
		OnlyAlive bool `cli:"-o, --only-alive"`
		Shallow   bool `cli:"-s, --shallow"`
	} `cli:"export"`

	Import struct {
		IgnoreDestroyed bool `cli:"-I, --ignore-destroyed"`
		IgnoreDeleted   bool `cli:"-i, --ignore-deleted"`
		Shallow         bool `cli:"-s, --shallow"`
	} `cli:"import"`

	Move struct {
		Recurse bool `cli:"-R, -r, --recurse"`
		Force   bool `cli:"-f, --force"`
		Deep    bool `cli:"-d, --deep"`
	} `cli:"move, rename, mv"`

	Copy struct {
		Recurse bool `cli:"-R, -r, --recurse"`
		Force   bool `cli:"-f, --force"`
		Deep    bool `cli:"-d, --deep"`
	} `cli:"copy, cp"`

	Gen struct {
		Policy string `cli:"-p, --policy"`
		Length int    `cli:"-l, --length"`
	} `cli:"gen, auto, generate"`

	SSH     struct{} `cli:"ssh"`
	RSA     struct{} `cli:"rsa"`
	DHParam struct{} `cli:"dhparam, dhparams, dh"`
	Prompt  struct{} `cli:"prompt"`
	Vault   struct{} `cli:"vault!"`
	Fmt     struct{} `cli:"fmt"`

	Curl struct {
		DataOnly bool `cli:"--data-only"`
	} `cli:"curl"`

	UUID   struct{} `cli:"uuid"`
	Option struct{} `cli:"option"`

	X509 struct {
		Validate struct {
			CA         bool     `cli:"-A, --ca"`
			SignedBy   string   `cli:"-i, --signed-by"`
			NotRevoked bool     `cli:"-R, --not-revoked"`
			Revoked    bool     `cli:"-r, --revoked"`
			NotExpired bool     `cli:"-E, --not-expired"`
			Expired    bool     `cli:"-e, --expired"`
			Name       []string `cli:"-n, --for"`
			Bits       []int    `cli:"-b, --bits"`
		} `cli:"validate, check"`

		Issue struct {
			CA           bool     `cli:"-A, --ca"`
			Subject      string   `cli:"-s, --subj, --subject"`
			Bits         int      `cli:"-b, --bits"`
			SignedBy     string   `cli:"-i, --signed-by"`
			Name         []string `cli:"-n, --name"`
			TTL          string   `cli:"-t, --ttl"`
			KeyUsage     []string `cli:"-u, --key-usage"`
			SigAlgorithm string   `cli:"-l, --sig-algorithm"`
		} `cli:"issue"`

		Revoke struct {
			SignedBy string `cli:"-i, --signed-by"`
		} `cli:"revoke"`

		Renew struct {
			Subject      string   `cli:"-s, --subj, --subject"`
			Name         []string `cli:"-n, --name"`
			SignedBy     string   `cli:"-i, --signed-by"`
			TTL          string   `cli:"-t, --ttl"`
			KeyUsage     []string `cli:"-u, --key-usage"`
			SigAlgorithm string   `cli:"-l, --sig-algorithm"`
		} `cli:"renew"`

		Reissue struct {
			Subject      string   `cli:"-s, --subj, --subject"`
			Name         []string `cli:"-n, --name"`
			Bits         int      `cli:"-b, --bits"`
			SignedBy     string   `cli:"-i, --signed-by"`
			TTL          string   `cli:"-t, --ttl"`
			KeyUsage     []string `cli:"-u, --key-usage"`
			SigAlgorithm string   `cli:"-l, --sig-algorithm"`
		} `cli:"reissue"`

		Show struct {
		} `cli:"show"`

		CRL struct {
			Renew bool `cli:"--renew"`
		} `cli:"crl"`
	} `cli:"x509"`
}

func main() {
	var opt Options
	opt.Gen.Policy = "a-zA-Z0-9"

	opt.Clobber = true

	opt.X509.Issue.Bits = 4096

	opt.Init.Persist = true
	opt.Rekey.Persist = true

	opt.Target.Strongbox = true

	go Signals()

	r := NewRunner()

	registerHelpCommands(r, &opt)
	registerTargetCommands(r, &opt)
	registerAuthCommands(r, &opt)
	registerSecretCommands(r, &opt)
	registerTreeCommands(r, &opt)
	registerMigrationCommands(r, &opt)
	registerGenerateCommands(r, &opt)
	registerUtilsCommands(r, &opt)
	registerX509Commands(r, &opt)
	registerAdminCommands(r, &opt)

	env.Override(&opt)
	p, err := cli.NewParser(&opt, os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "@R{!! %s}\n", err)
		os.Exit(1)
	}

	if opt.Version {
		r.Execute("version")
		return
	}
	if opt.Help { //-h was given as a global arg
		r.Execute("help")
		return
	}

	for p.Next() {
		opt.SkipIfExists = !opt.Clobber

		if opt.Version {
			r.Execute("version")
			return
		}

		if p.Command == "" { //No recognized command was found
			r.Execute("help")
			return
		}

		if opt.Help { // -h or --help was given after a command
			r.Execute("help", p.Command)
			continue
		}

		os.Unsetenv("VAULT_SKIP_VERIFY")
		os.Unsetenv("SAFE_SKIP_VERIFY")
		if opt.Insecure {
			os.Setenv("VAULT_SKIP_VERIFY", "1")
			os.Setenv("SAFE_SKIP_VERIFY", "1")
		}

		defer rc.Cleanup()
		err = r.Execute(p.Command, p.Args...)
		if err != nil {
			if strings.HasPrefix(err.Error(), "USAGE") {
				fmt.Fprintf(os.Stderr, "@Y{%s}\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "@R{!! %s}\n", err)
			}
			os.Exit(1)
		}
	}

	//If there were no args given, the above loop that would try to give help
	// doesn't execute at all, so we catch it here.
	if p.Command == "" {
		r.Execute("help")
	}

	if err = p.Error(); err != nil {
		fmt.Fprintf(os.Stderr, "@R{!! %s}\n", err)
		os.Exit(1)
	}
}
