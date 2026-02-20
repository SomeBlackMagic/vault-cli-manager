package app

import (
	"bytes"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Runner", func() {
	Describe("NewRunner", func() {
		It("creates a non-nil Runner", func() {
			r := NewRunner()
			Expect(r).ToNot(BeNil())
		})

		It("initializes Handlers map", func() {
			r := NewRunner()
			Expect(r.Handlers).ToNot(BeNil())
			Expect(len(r.Handlers)).To(Equal(0))
		})

		It("initializes Topics map", func() {
			r := NewRunner()
			Expect(r.Topics).ToNot(BeNil())
			Expect(len(r.Topics)).To(Equal(0))
		})
	})

	Describe("Dispatch", func() {
		It("registers a handler for the given command", func() {
			r := NewRunner()
			r.Dispatch("test", &Help{Summary: "Test cmd"}, func(cmd string, args ...string) error {
				return nil
			})
			Expect(r.Handlers).To(HaveKey("test"))
		})

		It("registers the help topic for the command with trimmed description", func() {
			r := NewRunner()
			r.Dispatch("test", &Help{Summary: "Test cmd", Description: "\nsome desc\n"}, func(cmd string, args ...string) error { return nil })
			Expect(r.Topics).To(HaveKey("test"))
			Expect(r.Topics["test"].Description).To(Equal("some desc"))
		})

		It("does not register help topic for HiddenCommand type", func() {
			r := NewRunner()
			r.Dispatch("hidden", &Help{Summary: "Hidden", Type: HiddenCommand}, func(cmd string, args ...string) error { return nil })
			Expect(r.Topics).ToNot(HaveKey("hidden"))
			Expect(r.Handlers).To(HaveKey("hidden"))
		})

		It("does not register help topic when help is nil", func() {
			r := NewRunner()
			r.Dispatch("nohelp", nil, func(cmd string, args ...string) error { return nil })
			Expect(r.Topics).ToNot(HaveKey("nohelp"))
			Expect(r.Handlers).To(HaveKey("nohelp"))
		})
	})

	Describe("Execute", func() {
		It("calls the correct handler and returns its result", func() {
			r := NewRunner()
			var receivedCmd string
			var receivedArgs []string
			r.Dispatch("test", nil, func(cmd string, args ...string) error {
				receivedCmd = cmd
				receivedArgs = args
				return nil
			})
			err := r.Execute("test", "arg1", "arg2")
			Expect(err).ToNot(HaveOccurred())
			Expect(receivedCmd).To(Equal("test"))
			Expect(receivedArgs).To(Equal([]string{"arg1", "arg2"}))
		})

		It("returns an error for an unknown command", func() {
			r := NewRunner()
			err := r.Execute("nonexistent")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unknown command"))
			Expect(err.Error()).To(ContainSubstring("nonexistent"))
		})

		It("propagates errors returned by the handler", func() {
			r := NewRunner()
			r.Dispatch("fail", nil, func(cmd string, args ...string) error {
				return fmt.Errorf("handler error")
			})
			err := r.Execute("fail")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("handler error"))
		})
	})

	Describe("Help", func() {
		var r *Runner

		BeforeEach(func() {
			r = NewRunner()
			r.Dispatch("get", &Help{
				Summary:     "Retrieve a secret",
				Usage:       "safe get PATH",
				Description: "Gets a secret from the Vault",
				Type:        NonDestructiveCommand,
			}, func(cmd string, args ...string) error { return nil })
			r.Dispatch("set", &Help{
				Summary: "Create or update a secret",
				Usage:   "safe set PATH k=v",
				Type:    DestructiveCommand,
			}, func(cmd string, args ...string) error { return nil })
		})

		Context("with topic 'commands'", func() {
			It("lists all registered commands", func() {
				buf := &bytes.Buffer{}
				r.Help(buf, "commands")
				output := buf.String()
				Expect(output).To(ContainSubstring("get"))
				Expect(output).To(ContainSubstring("set"))
				Expect(output).To(ContainSubstring("Valid commands"))
			})
		})

		Context("with a known command topic", func() {
			It("displays the command summary and description", func() {
				buf := &bytes.Buffer{}
				r.Help(buf, "get")
				output := buf.String()
				Expect(output).To(ContainSubstring("Retrieve a secret"))
				Expect(output).To(ContainSubstring("safe get PATH"))
				Expect(output).To(ContainSubstring("Gets a secret from the Vault"))
			})
		})

		Context("with a known command topic that has no description", func() {
			It("displays the summary and usage only", func() {
				buf := &bytes.Buffer{}
				r.Help(buf, "set")
				output := buf.String()
				Expect(output).To(ContainSubstring("Create or update a secret"))
				Expect(output).To(ContainSubstring("safe set PATH k=v"))
			})
		})

		Context("with an unknown topic", func() {
			It("cannot be tested because Help calls os.Exit(1)", func() {
				Skip("Help calls os.Exit(1) for unknown topics - cannot test without subprocess")
			})
		})
	})

	Describe("HelpTopic", func() {
		It("registers a help topic without a handler", func() {
			r := NewRunner()
			r.HelpTopic("envvars", "\nEnvironment variables info\n")
			Expect(r.Topics).To(HaveKey("envvars"))
			Expect(r.Topics["envvars"].Description).To(Equal("Environment variables info"))
		})
	})
})
