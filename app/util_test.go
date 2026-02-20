package app

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Util", func() {
	Describe("Duration", func() {
		It("parses hours with lowercase h", func() {
			d, err := Duration("10h")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(10 * time.Hour))
		})

		It("parses hours with uppercase H", func() {
			d, err := Duration("10H")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(10 * time.Hour))
		})

		It("parses days with lowercase d", func() {
			d, err := Duration("5d")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(5 * 24 * time.Hour))
		})

		It("parses days with uppercase D", func() {
			d, err := Duration("5D")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(5 * 24 * time.Hour))
		})

		It("parses months (30 days) with lowercase m", func() {
			d, err := Duration("3m")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(3 * 30 * 24 * time.Hour))
		})

		It("parses months with uppercase M", func() {
			d, err := Duration("3M")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(3 * 30 * 24 * time.Hour))
		})

		It("parses years (365 days) with lowercase y", func() {
			d, err := Duration("2y")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(2 * 365 * 24 * time.Hour))
		})

		It("parses years with uppercase Y", func() {
			d, err := Duration("2Y")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(2 * 365 * 24 * time.Hour))
		})

		It("parses single unit value", func() {
			d, err := Duration("1h")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(time.Hour))
		})

		It("parses zero value", func() {
			d, err := Duration("0h")
			Expect(err).ToNot(HaveOccurred())
			Expect(d).To(Equal(time.Duration(0)))
		})

		It("returns error for invalid format - no unit", func() {
			_, err := Duration("10")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unrecognized time spec"))
		})

		It("returns error for invalid format - letters only", func() {
			_, err := Duration("abc")
			Expect(err).To(HaveOccurred())
		})

		It("returns error for empty string", func() {
			_, err := Duration("")
			Expect(err).To(HaveOccurred())
		})

		It("returns error for invalid unit", func() {
			_, err := Duration("10x")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Uniq", func() {
		It("deduplicates preserving order", func() {
			result := Uniq([]string{"a", "b", "a", "c", "b"})
			Expect(result).To(Equal([]string{"a", "b", "c"}))
		})

		It("returns empty slice for empty input", func() {
			result := Uniq([]string{})
			Expect(result).To(BeEmpty())
		})

		It("returns the same slice for no duplicates", func() {
			result := Uniq([]string{"x", "y", "z"})
			Expect(result).To(Equal([]string{"x", "y", "z"}))
		})

		It("handles single element", func() {
			result := Uniq([]string{"only"})
			Expect(result).To(Equal([]string{"only"}))
		})

		It("handles all duplicates", func() {
			result := Uniq([]string{"a", "a", "a"})
			Expect(result).To(Equal([]string{"a"}))
		})

		It("handles nil input", func() {
			result := Uniq(nil)
			Expect(result).To(BeEmpty())
		})
	})
})
