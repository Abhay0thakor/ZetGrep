package classifier

import (
	"github.com/projectdiscovery/utils/ml/naive_bayes"
)

type Classifier struct {
	nb *naive_bayes.NaiveBayesClassifier
}

func New(smoothing float64) *Classifier {
	return &Classifier{
		nb: naive_bayes.New(smoothing),
	}
}

func (c *Classifier) Fit(data map[string][]string) {
	c.nb.Fit(data)
}

func (c *Classifier) Classify(text string) string {
	return c.nb.Classify(text)
}

func DefaultClassifier() *Classifier {
	c := New(1.1)
	c.Fit(map[string][]string{
		"high-interest": {"AKIA", "SECRET", "PRIVATE KEY", "mongodb://", "mysql://", "token", "password", "auth"},
		"low-interest":  {"//", "import", "func", "<html>", "package", "var", "const", "class"},
	})
	return c
}
