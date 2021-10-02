// One big mistake, don't worry about it
package khtml

var startingBytes = []byte("<!DOCTYPE html")

type Html struct {
	buff    []byte
	stack   []string
	depth   int
	content bool
}

func (h *Html) Doctype(tp string) *Html {
	h.buff = append(h.buff, "<!DOCTYPE "...)
	h.buff = append(h.buff, tp...)
	return h
}

func (h *Html) Style(path string) *Html {
	return h.Tag("link", "rel", "stylesheet", "href", path).End()
}

func (h *Html) Script(path string) *Html {
	return h.Tag("script", "src", path).End()
}

func (h *Html) SimpleTag(s string) *Html {
	return h.Text("<" + s + ">")
}

func (h *Html) Text(s string) *Html {
	if h.depth == 0 {
		panic("html: cannot set content in root")
	}

	if !h.content {
		h.content = true
		h.buff = append(h.buff, '>')
	}
	h.buff = append(h.buff, s...)
	return h
}

func (h *Html) Wrap(other *Html) *Html {
	if !h.content {
		h.content = true
		h.buff = append(h.buff, '>')
	}
	h.buff = other.Close(h.buff)
	return h
}

func (h *Html) Tag(name string, attrs ...string) *Html {
	h.stack = append(h.stack, name)
	if !h.content {
		h.buff = append(h.buff, '>')
	}
	h.content = false

	h.buff = append(h.buff, '<')
	h.buff = append(h.buff, name...)
	h.depth++

	l := len(attrs) - 1
	for i := 0; i < l; i += 2 {
		h.buff = append(h.buff, ' ')
		h.buff = append(h.buff, attrs[i]...)
		h.buff = append(h.buff, '=')
		h.buff = append(h.buff, '"')
		h.buff = append(h.buff, attrs[i+1]...)
		h.buff = append(h.buff, '"')
	}

	return h
}

func (h *Html) End() *Html {
	if !h.content {
		h.buff = append(h.buff, '>')
	}
	h.content = false

	h.depth--

	h.buff = append(h.buff, "</"...)
	h.buff = append(h.buff, h.stack[len(h.stack)-1]...)
	h.stack = h.stack[:len(h.stack)-1]

	return h
}

func (h *Html) Close(buff []byte) []byte {
	for len(h.stack) != 0 {
		h.End()
	}
	h.buff = append(h.buff, '>')
	res := append(buff, h.buff...)
	h.buff = append(h.buff[:0], startingBytes...)
	return res
}
