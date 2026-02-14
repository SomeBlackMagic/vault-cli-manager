package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UI", func() {
	Describe("parseKeyVal", func() {
		Context("with key=value syntax", func() {
			It("parses key and value from 'k=v'", func() {
				key, val, prompt, err := parseKeyVal("mykey=myvalue", true)
				Expect(err).ToNot(HaveOccurred())
				Expect(key).To(Equal("mykey"))
				Expect(val).To(Equal("myvalue"))
				Expect(prompt).To(BeFalse())
			})

			It("parses key with empty value from 'k='", func() {
				key, val, prompt, err := parseKeyVal("mykey=", true)
				Expect(err).ToNot(HaveOccurred())
				Expect(key).To(Equal("mykey"))
				Expect(val).To(Equal(""))
				Expect(prompt).To(BeFalse())
			})

			It("handles value containing '=' characters", func() {
				key, val, prompt, err := parseKeyVal("key=a=b=c", true)
				Expect(err).ToNot(HaveOccurred())
				Expect(key).To(Equal("key"))
				Expect(val).To(Equal("a=b=c"))
				Expect(prompt).To(BeFalse())
			})

			It("returns correct values when quiet is false", func() {
				key, val, prompt, err := parseKeyVal("k=v", false)
				Expect(err).ToNot(HaveOccurred())
				Expect(key).To(Equal("k"))
				Expect(val).To(Equal("v"))
				Expect(prompt).To(BeFalse())
			})
		})

		Context("with key@file syntax", func() {
			var tmpDir string
			var tmpFile string

			BeforeEach(func() {
				var err error
				tmpDir, err = ioutil.TempDir("", "ui-test")
				Expect(err).ToNot(HaveOccurred())
				tmpFile = filepath.Join(tmpDir, "testfile.txt")
				err = ioutil.WriteFile(tmpFile, []byte("file contents here"), 0644)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				os.RemoveAll(tmpDir)
			})

			It("reads file contents for 'key@filepath'", func() {
				key, val, prompt, err := parseKeyVal("mykey@"+tmpFile, true)
				Expect(err).ToNot(HaveOccurred())
				Expect(key).To(Equal("mykey"))
				Expect(val).To(Equal("file contents here"))
				Expect(prompt).To(BeFalse())
			})

			It("returns an error for 'key@' with no filename", func() {
				key, _, prompt, err := parseKeyVal("mykey@", true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("No file specified"))
				Expect(key).To(Equal("mykey"))
				Expect(prompt).To(BeTrue())
			})

			It("returns an error when file does not exist", func() {
				_, _, _, err := parseKeyVal("mykey@/nonexistent/file.txt", true)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Failed to read"))
			})
		})

		Context("with bare key (no = or @)", func() {
			It("signals that a prompt is needed", func() {
				key, val, prompt, err := parseKeyVal("barekey", true)
				Expect(err).ToNot(HaveOccurred())
				Expect(key).To(Equal("barekey"))
				Expect(val).To(Equal(""))
				Expect(prompt).To(BeTrue())
			})
		})

		Context("edge cases", func() {
			It("handles an empty string", func() {
				key, val, prompt, err := parseKeyVal("", true)
				Expect(err).ToNot(HaveOccurred())
				Expect(key).To(Equal(""))
				Expect(val).To(Equal(""))
				Expect(prompt).To(BeTrue())
			})
		})
	})
})
