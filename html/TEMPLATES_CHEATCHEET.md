# Golang Templates Cheatsheet

The Go standard library provides a set of packages to generate output. The [text/template](https://archive.is/o/2HksZ/https://golang.org/pkg/text/template/) package implements templates for generating text output, while the [html/template](https://archive.is/o/2HksZ/https://golang.org/pkg/html/template/) package implements templates for generating HTML output that is safe against certain attacks. Both packages use the same interface but the following examples of the core features are directed towards HTML applications.

* * *

## Table of Contents

*   [Parsing and Creating Templates](#parsing-and-creating-templates)
*   [Executing Templates](#executing-templates)
*   [Template Encoding and HTML](#template-encoding-and-html)
*   [Template Variables](#template-variables)
*   [Template Actions](#template-actions)
*   [Template Functions](#template-functions)
*   [Template Comparison Functions](#template-comparison-functions)
*   [Nested Templates and Layouts](#nested-templates-and-layouts)
*   [Templates Calling Functions](#templates-calling-functions)

* * *

## Parsing and Creating Templates

#### Naming Templates

There is no defined file extension for Go templates. One of the most popular is `.tmpl` supported by vim-go and [referenced in the text/template godocs](https://archive.is/o/2HksZ/golang.org/pkg/text/template/%23example_Template_helpers). The extension `.gohtml` supports syntax highlighting in both Atom and GoSublime editors. Finally analysis of large Go codebases finds that `.tpl` is often used by developers. While the extension is not important it is still good to be consistent within a project for clarity.

* * *

#### Creating a Template

`tpl, err := template.Parse(filename)` will get the template at filename and store it in tpl. tpl can then be executed to show the template.

* * *

#### Parsing Multiple Templates

`template.ParseFiles(filenames)` takes a list of filenames and stores all templates. `template.ParseGlob(pattern)` will find all templates matching the pattern and store the templates.

* * *

## Executing Templates

#### Execute a Single Template

Once a template has been parsed there are two options to execute them. A single template `tpl` can be executed using `tpl.Execute(io.Writer, data)`. The content of tpl will be written to the io.Writer. Data is an interface passed to the template that will be useable in the template.

* * *

#### Executing a Named Template

`tpl.ExecuteTemplate(io.Writer, name, data)` works the same as execute but allows for a string name of the template the user wants to execute.

* * *

## Template Encoding and HTML

#### Contextual Encoding

Go’s html/template package does encoding based on the context of the code. As a result, html/template encodes any characters that need encoding to be rendered correctly.

For example the < and > in `"<h1>A header!</h1>"` will be encoded as `&lt;h1&gt;A header!&lt;/h1&gt;` .

Type `template.HTML` can be used to skip encoding by telling Go the string is safe. `template.HTML("<h1>A Safe header</h1>")` will then be `<h1>A Safe header</h1>` . Using this type with user input is dangerous and leaves the application vulnerable.

The go `html/template` package is aware of attributes within the template and will encode values differently based on the attribute.

Go templates can also be used with javascript. Structs and maps will be expanded into JSON objects and quotes will be added to strings for use in function parameters and as variable values.



```go
    // Go
    type Cat struct {
    	Name string
    	Age int
    }

    kitten := Cat{"Sam", 12}
```
```html
    // Template
    <script>
    	var cat = {{.kitten}}
    </script>
```
```js
    // Javascript
    var cat = {"Name":"Sam", "Age" 12}
```

* * *

#### Safe Strings and HTML Comments

The `html/template` package will remove any comments from a template by default. This can cause issues when comments are necessary such as detecting internet explorer.

```html
    <!--[if IE]>
    Place content here to target all Internet Explorer users.
    <![endif]-->
```

We can use the Custom Functions method (Globally) to create a function that returns html preserving comments. Define a function `htmlSafe` in the FuncMap of the template.

```go
    testTemplate, err = template.New("hello.gohtml").Funcs(template.FuncMap{
    	"htmlSafe": func(html string) template.HTML {
    		return template.HTML(html)
        },
    }).ParseFiles("hello.gohtml")
```

This function takes a string and produces the unaltered HTML code. This function can be used in a template like so to preserve the comments `<!--[if IE 6]>` and `<![endif]-->` :

```go
    {{htmlSafe "<!--[if IE 6]>" }}
    <meta http-equiv="Content-Type" content="text/html; charset=Unicode">  
    {{ htmlSafe "<![endif]-->" }}
```

* * *

## Template Variables

#### The dot character (.)

A template variable can be a boolean, string, character, integer, floating-point, imaginary, or complex constant in Go syntax. Data passed to the template can be accessed using dot `{{ . }}`.

If the data is a complex type then it’s fields can be accessed using the dot with the field name `{{ .FieldName }}`.

Dots can be chained together if the data contains multiple complex structures. `{{ .Struct.StructTwo.Field }}`

* * *

#### Variables in Templates

Data passed to the template can be saved in a variable and used throughout the template. `{{$number := .}}` We use the `$number` to create a variable then initialize it with the value passed to the template. To use the variable we call it in the template with `{{$number}}`.

```go
    {{$number := .}}
    <h1> It is day number {{$number}} of the month </h1>
```
```go
    var tpl *template.Template

    tpl = template.Must(template.ParseFiles("templateName"))

    err := tpl.ExecuteTemplate(os.Stdout, "templateName", 23)
```
In this example we pass 23 to the template and stored in the `$number` variable which can be used anywhere in the template

* * *

## Template Actions


#### If/Else Statements

Go templates support if/else statements like many programming languages. We can use the if statement to check for values, if it doesn’t exist we can use an else value. The empty values are false, 0, any nil pointer or interface value, and any array, slice, map, or string of length zero.

```html
    <h1>Hello, {{if .Name}} {{.Name}} {{else}} Anonymous {{end}}!</h1>  
```

If .Name exists then `Hello, Name` will be printed (replaced with the name value) otherwise it will print `Hello, Anonymous`.

Templates also provide the else if statment `{{else if .Name2 }}` which can be used to evaluate other options after an if.

* * *

#### Removing Whitespace

Adding different values to a template can add various amounts of whitespace. We can either change our template to better handle it, by ignoring or minimizing effects, or we can use the minus sign `-` within out template.

`<h1>Hello, {{if .Name}} {{.Name}} {{- else}} Anonymous {{- end}}!</h1>`

Here we are telling the template to remove all spaces between the `Name` variable and whatever comes after it. We are doing the same with the end keyword. This allows us to have whitespace within the template for easier reading but remove it in production.

* * *

#### Range Blocks

Go templates have a `range` keyword to iterate over all objects in a structure. Suppose we had the Go structures:

```go
    type Item struct {
    	Name  string
    	Price int
    }

    type ViewData struct {
    	Name  string
    	Items []Item
    }
```

We have an Item, with a name and price, then a ViewData which is the structure sent to the template. Consider the template containing the following:

```html
    {{range .Items}}
      <div class="item">
        <h3 class="name">{{.Name}}</h3>
        <span class="price">${{.Price}}</span>
      </div>
    {{end}}
```

For each Item in the range of Items (in the ViewData structure) get the Name and Price of that item and create html for each Item automatically. Within a range each Item becomes the `{{.}}` and the item properties therefore become `{{.Name}}` or `{{.Price}}` in this example.

* * *

## Template Functions

The template package provides a list of predefined global functions. Below are some of the most used.

* * *

#### Indexing structures in Templates

If the data passed to the template is a map, slice, or array it can be indexed from the template. We use `{{index x number}}` where index is the keyword, x is the data and number is a integer for the index value. If we had `{{index names 2}}` it is equivalent to `names[2]`. We can add more integers to index deeper into data. `{{index names 2 3 4}}` is equivalent to `names[2][3][4]`.

```html
    <body>
        <h1> {{index .FavNums 2 }}</h1>
    </body>
```

```go
    type person struct {
    	Name    string
    	FavNums []int
    }

    func main() {

    	tpl := template.Must(template.ParseGlob("*.gohtml"))
    	tpl.Execute(os.Stdout, &person{"Curtis", []int{7, 11, 94}})
    }
```

This code example passes a person structure and gets the 3rd favourite number from the FavNums slice.

* * *

#### The `and` Function

The and function returns the boolean AND of its arguments by returning the first empty argument or the last argument. `and x y` behaves logically as `if x then y else x` . Consider the following go code

```go
    type User struct {  
      Admin bool
    }

    type ViewData struct {  
      *User
    }
```

Pass a ViewData with a User that has Admin set true to the following template

```go

    {{if and .User .User.Admin}}
      You are an admin user!
    {{else}}
      Access denied!
    {{end}}
```

The result will be `You are an admin user!`. However if the ViewData did not include a *User object or Admin was set as false then the result will be `Access denied!`.

* * *

#### The `or` Function

The or function operates similarly to the and function however will stop at the first true. `or x y` is equivalent to `if x then x else y` so y will never be evaluated if x is not empty.

* * *

#### The `not` Function

The not function returns the boolean negation of the argument.

```go
    {{ if not .Authenticated}}
      Access Denied!
    {{ end }}
```

* * *

## Template Comparison Functions


#### Comparisons

The `html/template` package provides a variety of functions to do comparisons between operators. The operators may only be basic types or named basic types such as `type Temp float32` Remember that template functions take the form `{{ function arg1 arg2 }}`.

*   `eq` Returns the result of arg1 == arg2
*   `ne` Returns the result of arg1 != arg2
*   `lt` Returns the result of arg1 < arg2
*   `le` Returns the result of arg1 <= arg2
*   `gt` Returns the result of arg1 > arg2
*   `ge` Returns the result of arg1 >= arg2

Of special note `eq` can be used with two or more arguments by comparing all arguments to the first. `{{ eq arg1 arg2 arg3 arg4}}` will result in the following logical expression:

`arg1==arg2 || arg1==arg3 || arg1==arg4`

* * *

## Nested Templates and Layouts


#### Nesting Templates

Nested templates can be used for parts of code frequently used across templates, a footer or header for example. Rather than updating each template separately we can use a nested template that all other templates can use. You can define a template as follows:

```go
    {{define "footer"}}
    <footer> 
    	<p>Here is the footer</p>
    </footer>
    {{end}}
```

A template named “footer” is defined which can be used in other templates like so to add the footer template content into the other template:

```go
    {{template "footer"}}
```

* * *

#### Passing Variables between Templates

The `template` action used to include nested templates also allows a second parameter to pass data to the nested template.

```html
    // Define a nested template called header
    {{define "header"}}
    	<h1>{{.}}</h1>
    {{end}}

    // Call template and pass a name parameter
    {{range .Items}}
      <div class="item">
        {{template "header" .Name}}
        <span class="price">${{.Price}}</span>
      </div>
    {{end}}
```

We use the same range to loop through Items as before but we pass the name to the header template each time in this simple example.

* * *

#### Creating Layouts

Glob patterns specify sets of filenames with wildcard characters. The `template.ParseGlob(pattern string)` function will parse all templates that match the string pattern. `template.ParseFiles(files...)` can also be used with a list of file names.

The templates are named by default based on the base names of the argument files. This mean `views/layouts/hello.gohtml` will have the name `hello.gohtml` . If the template has a ``{{define “templateName”}}` within it then that name will be usable.

A specific template can be executed using `t.ExecuteTemplate(w, "templateName", nil)` . `t` is an object of type Template, `w` is type io.Writer such as an `http.ResponseWriter`, Then there is the name of the template to execute, and finally passing any data to the template, in this case a nil value.

Example main.go file

```go
    // Omitted imports & package

    var LayoutDir string = "views/layouts"  
    var bootstrap *template.Template

    func main() {
    	var err error
    	bootstrap, err = template.ParseGlob(LayoutDir + "/*.gohtml")
    	if err != nil {
    		panic(err)
    	}

    	http.HandleFunc("/", handler)
    	http.ListenAndServe(":8080", nil)
    }

    func handler(w http.ResponseWriter, r *http.Request) {
    	bootstrap.ExecuteTemplate(w, "bootstrap", nil)
    }
```

All `.gohtml` files are parsed in main. When route `/` is reached the template defined as `bootstrap` is executed using the handler function.

Example views/layouts/bootstrap.gohtml file

```html
    {{define "bootstrap"}}
    <!DOCTYPE html>  
    <html lang="en">  
      <head>
        <title>Go Templates</title>
        <link href="//maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css" 
    	rel="stylesheet">
      </head>
      <body>
        <div class="container-fluid">
          <h1>Filler header</h1>
    	  <p>Filler paragraph</p>
        </div>
        <!-- jquery & Bootstrap JS -->
        <script src="//ajax.googleapis.com/ajax/libs/jquery/1.11.3/jquery.min.js"  
        </script>
        <script src="//maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js">
        </script>
      </body>
    </html>  
    {{end}}
```

## Templates Calling Functions


#### Function Variables (calling struct methods)

We can use templates to call the methods of objects in the template to return data. Consider the User struct with the following method.

```go
    type User struct {  
      ID    int
      Email string
    }

    func (u User) HasPermission(feature string) bool {  
      if feature == "feature-a" {
        return true
      } else {
        return false
      }
    }
```

When a type User has been passed to the template we can then call this method from the template.

```html
    {{if .User.HasPermission "feature-a"}}
      <div class="feature">
        <h3>Feature A</h3>
        <p>Some other stuff here...</p>
      </div>
    {{else}}
      <div class="feature disabled">
        <h3>Feature A</h3>
        <p>To enable Feature A please upgrade your plan</p>
      </div>
    {{end}}
```

The template checks if the User HasPermission for the feature and renders depending on the result.

* * *

#### Function Variables (call)

If the Method HasPermission has to change at times then the Function Variables (Methods) implementation may not fit the design. Instead a `HasPermission func(string) bool` attribute can be added on the `User` type. This can then have a function assigned to it at creation.

```go
    // Structs
    type ViewData struct {  
      User User
    }

    type User struct {  
      ID            int
      Email         string
      HasPermission func(string) bool
    }

    // Example of creating a ViewData
    vd := ViewData{
    		User: User{
    			ID:    1,
    			Email: "curtis.vermeeren@gmail.com",
    			// Create the HasPermission function
    			HasPermission: func(feature string) bool {
    				if feature == "feature-b" {
    					return true
    				}
    				return false
    			},
    		},
    	}

    // Executing the ViewData with the template
    err := testTemplate.Execute(w, vd)
```

We need to tell the Go template that we want to call this function so we must change the template from the Function Variables (Methods) implementation to do this. We use the `call` keyword supplied by the go `html/template` package. Changing the previous template to use `call` results in:

```html
    {{if (call .User.HasPermission "feature-b")}}
      <div class="feature">
        <h3>Feature B</h3>
        <p>Some other stuff here...</p>
      </div>
    {{else}}
      <div class="feature disabled">
        <h3>Feature B</h3>
        <p>To enable Feature B please upgrade your plan</p>
      </div>
    {{end}}
```

* * *

#### Custom Functions

Another way to call functions is to create custom functions with `template.FuncMap` . This method creates global methods that can be used throughout the entire application. FuncMap has type `map[string]interface{}` mapping a string, the function name, to a function. The mapped functions must have either a single return value, or two return values where the second has type error.

```go
    // Creating a template with function hasPermission
    testTemplate, err = template.New("hello.gohtml").Funcs(template.FuncMap{
        "hasPermission": func(user User, feature string) bool {
          if user.ID == 1 && feature == "feature-a" {
            return true
          }
          return false
        },
      }).ParseFiles("hello.gohtml")
```

Here the function to check if a user has permission for a feature is mapped to the string `"hasPermission"` and stored in the FuncMap. Note that the custom functions must be created before calling `ParseFiles()`

The function could be executed in the template as follows:

```go
    {{ if hasPermission .User "feature-a" }}
```

The `.User` and string `"feature-a"` are both passed to `hasPermission` as arguments.

* * *

#### Custom Functions (Globally)

The previous two methods of custom functions rely on `.User` being passed to the template. This works in many cases but in a large application passing too many objects to a template can become difficult to maintain across many templates. We can change the implementation of the custom function to work without the .User being passed.

Using a similar feature example as the other 2 sections first you would have to create a default `hasPermission` function and define it in the template’s function map.

```go
      testTemplate, err = template.New("hello.gohtml").Funcs(template.FuncMap{
        "hasPermission": func(feature string) bool {
          return false
        },
      }).ParseFiles("hello.gohtml")
```

This function could be placed in `main()` or somewhere that ensures the default `hasPermission` is created in the `hello.gohtml` function map. The default function just returns false but it defines the function and implementation that doesn’t require `User` .

Next a closure could be used to redefine the `hasPermission` function. It would use the `User` data available when it is created in a handler rather than having `User` data passed to it. Within the handler for the template you can redefine any functions to use the information available.

```go
    func handler(w http.ResponseWriter, r *http.Request) {
    	w.Header().Set("Content-Type", "text/html")

    	user := User{
    		ID:    1,
    		Email: "Curtis.vermeeren@gmail.com",
    	}
    	vd := ViewData{}
    	err := testTemplate.Funcs(template.FuncMap{
    		"hasPermission": func(feature string) bool {
    			if user.ID == 1 && feature == "feature-a" {
    				return true
    			}
    			return false
    		},
    	}).Execute(w, vd)
    	if err != nil {
    		http.Error(w, err.Error(), http.StatusInternalServerError)
    	}
    }
```

In this handler a `User` is created with ID and Email, Then a `ViewData` is created without passing the user to it. The `hasPermission` function is redefined using `user.ID` which is available when the function is created. `{{if hasPermission "feature-a"}}` can be used in a template without having to pass a `User` to the template as the User object in the handler is used instead.

* * *
