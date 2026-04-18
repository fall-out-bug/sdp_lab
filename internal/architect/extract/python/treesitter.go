package python

func TreeSitterSupported() bool {
	return false
}

func TreeSitterImportQuery() string {
	return `
(import_statement
  name: (dotted_name
    (identifier) @import.name))

(import_from_statement
  module_name: (dotted_name
    (identifier) @from.module)
  name: (dotted_name
    (identifier) @from.import))
`
}

func TreeSitterFrameworkQuery() string {
	return `
(decorator
  (identifier) @decorator.name
  (#eq? @decorator.name "route")
  (attribute
    (identifier) @app.object
    (#eq? @app.object "app" "application" "api")))

(expression_statement
  (assignment
    left: (identifier) @app.var
    right: (call
      function: (identifier) @app.type
      (#eq? @app.type "FastAPI" "APIRouter"))))

(class_definition
  name: (identifier) @model.name
  superclasses: (argument_list
    (attribute
      (identifier) @model.base
      (#match? @model.base "^models\\.Model$|^Model$"))))
`
}
