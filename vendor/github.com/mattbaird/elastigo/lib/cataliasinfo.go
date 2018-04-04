package elastigo

import (
	"errors"
	"strings"
)

var ErrInvalidAliasLine = errors.New("Cannot parse aliasline")

//Create an AliasInfo from the string _cat/alias would produce
//EX: alias alias
//i   production production_20160405075824
func NewCatAliasInfo(aliasLine string) (catAlias *CatAliasInfo, err error) {
	split := strings.Fields(aliasLine)
	if len(split) < 2 {
		return nil, ErrInvalidAliasLine
	}
	catAlias = &CatAliasInfo{}
	catAlias.Name = split[0]
	catAlias.Index = split[1]
	return catAlias, nil
}

// Pull all the alias info from the connection
func (c *Conn) GetCatAliasInfo(pattern string) (catAliases []CatAliasInfo) {
	catAliases = make([]CatAliasInfo, 0)
	//force it to only show the fields we know about
	aliases, err := c.DoCommand("GET", "/_cat/aliases/"+pattern, nil, nil)
	if err == nil {
		aliasLines := strings.Split(string(aliases[:]), "\n")
		for _, alias := range aliasLines {
			ci, _ := NewCatAliasInfo(alias)
			if nil != ci {
				catAliases = append(catAliases, *ci)
			}
		}
	}
	return catAliases
}
