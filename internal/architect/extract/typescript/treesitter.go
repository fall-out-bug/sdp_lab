package typescript

// treesitter.go contains tree-sitter query patterns for TypeScript/JavaScript.
//
// NOTE: Current implementation uses regex/text parsing for import extraction.
// Tree-sitter queries are documented here for future migration to AST-based parsing,
// which will improve accuracy from the current 65-75% to 85-95%.
//
// Example tree-sitter queries for future implementation:
//
// ES Module Imports:
//   (import_statement
//     source: (string (string_fragment) @import.source))
//
// Named imports:
//   (import_statement
//     name: (import_clause
//       (named_imports
//         (import_specifier
//           name: (identifier) @import.name
//           alias: (identifier)? @import.alias))))
//
// Default imports:
//   (import_statement
//     name: (import_clause
//       (identifier) @import.default))
//
// CommonJS require:
//   (variable_declarator
//     name: (identifier)
//     value: (call_expression
//       function: (identifier) @req.fn
//       arguments: (arguments
//         (string (string_fragment) @req.source))))
//   (#eq? @req.fn "require"))
//
// Dynamic import:
//   (call_expression
//     function: (identifier) @imp.fn
//     arguments: (arguments
//       (string (string_fragment) @imp.source)))
//   (#eq? @imp.fn "import"))
//
// Re-exports:
//   (export_statement
//     source: (string (string_fragment) @export.source))
//
// NestJS decorators:
//   (decorator
//     (identifier) @decorator.name
//     (#match? @decorator.name "^(Module|Controller|Injectable)$"))
//
// React JSX elements:
//   (jsx_element
//     opening_element: (jsx_opening_element
//       name: (identifier) @jsx.tag))
//
// Next.js app directory detection:
//   (typescript) @file
//   (#match? @file "app/.*page\\.(tsx?|jsx?)$")
