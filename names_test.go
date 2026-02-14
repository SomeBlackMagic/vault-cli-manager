package main

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Names", func() {
	Describe("RandomName", func() {
		It("returns a non-empty string", func() {
			name := RandomName()
			Expect(name).ToNot(BeEmpty())
		})

		It("returns a string in adjective-noun format", func() {
			name := RandomName()
			parts := strings.SplitN(name, "-", 2)
			Expect(len(parts)).To(Equal(2))
			Expect(parts[0]).ToNot(BeEmpty())
			Expect(parts[1]).ToNot(BeEmpty())
		})

		It("uses an adjective from the Adjectives slice", func() {
			name := RandomName()
			parts := strings.SplitN(name, "-", 2)
			found := false
			for _, adj := range Adjectives {
				if adj == parts[0] {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("uses a noun from the Nouns slice", func() {
			name := RandomName()
			parts := strings.SplitN(name, "-", 2)
			found := false
			for _, noun := range Nouns {
				if noun == parts[1] {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
		})

		It("produces varying names over multiple calls", func() {
			names := make(map[string]bool)
			for i := 0; i < 50; i++ {
				names[RandomName()] = true
			}
			Expect(len(names)).To(BeNumerically(">", 1))
		})
	})

	Describe("Adjectives", func() {
		It("is a non-empty slice", func() {
			Expect(len(Adjectives)).To(BeNumerically(">", 0))
		})
	})

	Describe("Nouns", func() {
		It("is a non-empty slice", func() {
			Expect(len(Nouns)).To(BeNumerically(">", 0))
		})
	})
})
