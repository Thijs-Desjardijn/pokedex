package main

import (
	"testing"
)

func TestCleanInput(t *testing.T) {
	cases := []struct {
		input    string
		expected []string
	}{
		{
			input:    "  hello  world  ",
			expected: []string{"hello", "world"},
		},
		{
			input:    " HELLO WORLD THIS IS CHARMander ",
			expected: []string{"hello", "world", "this", "is", "charmander"},
		},
		{
			input:    "                                        hi",
			expected: []string{"hi"},
		},
		{
			input:    "",
			expected: []string{},
		},
		{
			input:    "hello\tworld",
			expected: []string{"hello", "world"},
		},
		{
			input:    "foo\nbar",
			expected: []string{"foo", "bar"},
		},
		{
			input:    "   one\t two\nthree   ",
			expected: []string{"one", "two", "three"},
		},
		{
			input:    "\n\t dragonite  \n\t",
			expected: []string{"dragonite"},
		},
		{
			input:    "pikachu\t\t\nbulbasaur  charmander",
			expected: []string{"pikachu", "bulbasaur", "charmander"},
		},
	}
	for _, c := range cases {
		actual := cleanInput(c.input)
		if len(actual) != len(c.expected) {
			t.Errorf("len of actual: %v and expected: %v don't match", actual, c.expected)
		}
		// if they don't match, use t.Errorf to print an error message
		// and fail the test
		for i := range actual {
			word := actual[i]
			expectedWord := c.expected[i]
			if word != expectedWord {
				t.Errorf("actual word: %v and expexted word: %v don't match", word, expectedWord)
			}
			// Check each word in the slice
			// if they don't match, use t.Errorf to print an error message
			// and fail the test
		}
	}
}
