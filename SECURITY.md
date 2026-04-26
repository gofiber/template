# Template Security Matrix

This repository now includes explicit XSS regression tests for every engine package. The existing per-engine GitHub Actions workflows already run `go test ./...`, so the new security tests run in CI without additional workflow changes.

## Audit Matrix

| Engine | Default escaping | Layout/embed behavior | Raw or unsafe primitives to watch | Helper/function trust model |
| --- | --- | --- | --- | --- |
| HTML | `html/template` contextual escaping | `embed` inserts already-rendered child content | Returning trusted `html/template` types such as `template.HTML`, `template.JS`, `template.URL`, or `template.CSS` bypasses escaping | Custom funcs are trusted code and must not return untrusted raw HTML |
| Ace | Compiles to `html/template` with contextual escaping | `embed` inserts already-rendered child content | Returning trusted `html/template` types bypasses escaping | Custom funcs are trusted code and must not return untrusted raw HTML |
| Amber | Compiles to `html/template` with contextual escaping | `embed()` inserts already-rendered child content | Returning trusted `html/template` types bypasses escaping | Custom funcs are trusted code and must not return untrusted raw HTML |
| Pug | Compiles to `html/template` with contextual escaping | `embed` inserts already-rendered child content | Returning trusted `html/template` types bypasses escaping | Custom funcs are trusted code and must not return untrusted raw HTML |
| Django | Auto-escape enabled by default | `embed` is wrapped with `pongo2.AsSafeValue` only after the child template has been rendered | `SetAutoEscape(false)`, `{% autoescape off %}`, and `safe`-style constructs disable escaping | Globals/helpers are trusted and should not manufacture safe HTML from untrusted input |
| Handlebars | Escapes HTML by default | Layout output is wrapped with `raymond.SafeString` only after the child template has been rendered | Triple-stash `{{{value}}}` and `raymond.SafeString` disable escaping | Helpers should return plain strings unless they intentionally produce trusted HTML |
| Mustache | Escapes HTML by default for `{{value}}` | Layouts render child output and inject it through `{{{embed}}}` | Triple-stash `{{{value}}}` and ampersand tags disable escaping | No helper API here, but any raw sections in templates become trust boundaries |
| Jet | Escapes HTML output by default | `embed()` renders the child template before insertion into the layout | Treat any helper or custom global that returns trusted markup as a bypass point | Globals and functions are trusted and must not emit raw HTML for untrusted input |
| Slim | `=` escapes output and `==` writes raw output | Layouts intentionally use `== embed` with already-rendered child content | `==` disables escaping anywhere it is used | Slim funcs are trusted and must avoid returning raw markup for untrusted input |

## Review Checklist

- Verify default escaping in body, attribute, URL, JavaScript-string, and CSS-adjacent contexts before documenting an engine as safe for that context.
- Only mark layout output as safe after the child template has been rendered and escaped.
- Never treat user-controlled `embed` values as trusted layout content.
- Treat helper output as untrusted by default unless the helper deliberately returns a library-specific safe type.
- Document every raw-output primitive near examples so users do not copy unsafe patterns accidentally.
