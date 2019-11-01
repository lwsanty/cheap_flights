package wikidata_test

import (
	"strings"
	"testing"

	"github.com/lwsanty/cheap_flights/wikidata"
	"github.com/stretchr/testify/require"
)

// TODO: more special cases like Rostov-on-Don
func TestTranslateCity(t *testing.T) {
	for _, cs := range []struct {
		srcLang  string
		src, dst string
		err      error
	}{
		{"en", "new york", "Нью-Йорк", nil},
		{"en", "kiev", "Киев", nil},
		{"en", "shoshosho", "", wikidata.NotFound},
	} {
		cs := cs
		t.Run(cs.src, func(t *testing.T) {
			actDst, err := wikidata.TranslateCity(cs.srcLang, cs.src)
			if cs.err != nil {
				require.Equal(t, cs.err, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, strings.ToLower(cs.dst), strings.ToLower(actDst))
		})
	}
}
