// Shared offline resources and CodeMirror behavior for Web and Qt WebEngine.
import { config } from 'md-editor-v3'
import hljs from 'highlight.js'
import { Prec } from '@codemirror/state'
import { keymap } from '@codemirror/view'
import { indentUnit, syntaxHighlighting, defaultHighlightStyle } from '@codemirror/language'
import { indentWithTab } from '@codemirror/commands'
import { markdownKeymap } from '@codemirror/lang-markdown'
import * as prettier from 'prettier/standalone'
import * as markdownParser from 'prettier/plugins/markdown'
import katex from 'katex'
import screenfull from 'screenfull'
import mermaid from 'mermaid'
import 'md-editor-v3/lib/style.css'
import 'md-editor-v3/lib/preview.css'
import 'highlight.js/styles/github.css'
import 'katex/dist/katex.min.css'
import './markdown.css'

config({
  editorExtensions: {
    iconfont: `${import.meta.env.BASE_URL}vendor/md-editor-icons.js`,
    highlight: { instance: hljs },
    prettier: { prettierInstance: prettier, parserMarkdownInstance: markdownParser },
    katex: { instance: katex },
    screenfull: { instance: screenfull },
    mermaid: { instance: mermaid },
  },
  codeMirrorExtensions: (_theme, extensions) => [
    ...extensions,
    indentUnit.of('  '),
    syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
    // Markdown Enter continues lists/quotes; Tab/Shift+Tab use CodeMirror indentation.
    Prec.highest(keymap.of([...markdownKeymap, indentWithTab])),
  ],
})
