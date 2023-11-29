package ratio

import "errors"

func (r *Ratio) Merge(r2 *Ratio) error {
	if len(r.CodeFiles) == 0 && len(r.TestFiles) == 0 {
		return errors.New("can not merge: CodeFiles and TestFiles are already deleted")
	}
	if len(r2.CodeFiles) == 0 && len(r2.TestFiles) == 0 {
		return errors.New("can not merge: CodeFiles and TestFiles are already deleted")
	}
	r.CodeFiles = uniqueFiles(append(r.CodeFiles, r2.CodeFiles...))
	r.TestFiles = uniqueFiles(append(r.TestFiles, r2.TestFiles...))
	code := 0
	test := 0
	for _, f := range r.CodeFiles {
		code += f.Code
	}
	for _, f := range r.TestFiles {
		test += f.Code
	}
	r.Code = code
	r.Test = test
	return nil
}

func uniqueFiles(in Files) Files {
	u := Files{}
	m := map[string]*File{}
	for _, f := range in {
		if v, ok := m[f.Path]; ok {
			v.Blanks = f.Blanks
			v.Code = f.Code
			v.Comments = f.Comments
			v.Lang = f.Lang
			v.Path = f.Path
			continue
		}
		u = append(u, f)
		m[f.Path] = f
	}
	return u
}
