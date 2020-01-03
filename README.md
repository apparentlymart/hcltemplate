# hcltemplate

`hcltemplate` is a filter program for transforming JSON objects into other
strings using the [HCL](https://hcl.readthedocs.io/) template language.

There are [precompiled releases](https://github.com/apparentlymart/hcltemplate/releases)
available for various platforms.

It expects to find a JSON object on its stdin and produces text on its stdout
using a template file given on the command line.

For example, given the following `input.json`:

```json
{
    "name": "Jacqueline"
}
```

...and the following `template.tmpl`:

```
Hello ${name}
```

We can render the template with the JSON input like this:

```
$ ./hcltemplate template.tmpl <input.json
Hello Jacqueline
```

`hcltemplate` is intended to be used in a pipeline with other software that
can produce JSON output. For a more complex example, we can use
[`terraform-config-inspect`](https://github.com/hashicorp/terraform-config-inspect),
which is a tool (and library) for inspecting the top-level objects in a
[Terraform](https://terraform.io/) configuration file.

Using the following template `terraform.tmpl`:

```
The module has the following input variables:
%{ for v in variables ~}
- ${v.name} (${v.type})
%{ endfor ~}

The module has the following output values:
%{ for o in outputs ~}
- ${o.name}
%{ endfor ~}
```

...we can pipe the JSON output from `terraform-config-inspect` into
`hcltemplate` like this:

```
$ terraform-config-inspect --json modules/network | hcltemplate terraform.tmpl 
The module has the following input variables:
- environment (string)

The module has the following output values:
- vpc_id
```

## Built-in Functions

The template language also includes a number of built-in functions that you can
call from expressions in your template, using the following syntax:

```
${upper("hello")}
```

The following functions are available:

| Function | Description |
| - | - |
| `abs(num)` | Returns the absolute value of the given number `num` |
| `can(expr)` | Returns `true` if the given expression evaluates without errors, or `false` if it produces at least one error. |
| `csvdecode(str)` | Attempts to interpret the given string as CSV-formatted data, returning a list of maps of strings. |
| `coalesce(vals...)` | Evaluates each of the given values in turn and returns the first one that isn't `null`. |
| `concat(lists...)` | Returns a single list that is the concatenation of all of the given lists, in order. |
| `convert(val, type)` | Converts the given `val` to the given `type`, where `type` is a special type expression like `string` or `list(string)`. |
| `format(pattern, vals...)` | "printf"-like formatting of arbitrary values using a pattern. |
| `formatdate(format, timestamp)` | Takes `timestamp` in RFC3339 format and reformats it using the given `format` string. For more information on the format syntax, see [Terraform's `formatdate` documentation](https://www.terraform.io/docs/configuration/functions/formatdate.html). |
| `int(num)` | Rounds `num` to be an integer, rounding towards zero. |
| `jsondecode(str)` | Parses the given string as JSON and returns an equivalent value. |
| `jsonencode(val)` | Produces a JSON-encoded string representation of the given value. |
| `length(val)` | Returns the length of the given collection (list, map, or set) or structure (object or tuple). |
| `lower(str)` | Returns a lowercase version of the given string. |
| `max(nums...)` | Returns the greatest value from the given numbers. |
| `min(nums...)` | Returns the smallest value from the given numbers. |
| `range(start, end, step)` | Returns a list of numbers. |
| `regex(pattern, str)` | Applies the given regular expression `pattern` to the given string `str`, returning information about a single match or an error if there is no match. |
| `regexall(pattern, str)` | Like `regex`, but returns a list of all matches rather than just a single match and will return an empty list if there are none. |
| `reverse(str)` | Returns a string with the characters from `str` given in reverse order. |
| `strlen(str)` | Returns the length of the given string, in Unicode characters (grapheme clusters). |
| `try(exprs...)` | Tries to evaluate each given expression in turn, returning the first one that produces no errors. |
| `upper(str)` | Returns an uppercase version of the given string. |

## Handling Unpredictable Data Structures

When formatting from arbitrary JSON it's common to need to deal with object
properties that might not always be set. By default, accessing a non-existent
property produces an error.

We can handle potentially-missing properties gracefully by using the `try`
function introduced in the previous section:

```
${try(obj.val, "default")}
```

The `try` function catches any errors that occur when evaluating one of the
expressions it is given, allowing it to continue to evaluate each expression
in turn until one succeeds.

In the above example, `obj.val` can fail if there is no property `"val"` or
if `obj` isn't an object at all, but `"default"` can never fail because it's
a constant and so we can use it as a fallback value for when the property
isn't set.

Another common variant is a property that can either be a single value or a
list of values:

```
${try(
  convert(maybe_list, list(string)),
  [convert(maybe_list, string)],
)}
```

The above example also uses `convert` as a type constraint assertion, because
`convert` will fail if the given value cannot be converted to the given type.
The above will first try to convert the value to a list of strings, returning
the result if it succeeds, or failing that it will try to convert the value
to a string and wrap it in a single-element list.

In that second example it's still possible for the entire expression to fail:
if `maybe_list` were an object value, for example, it would fail both conversion
to list _and_ conversion to string, and therefore `try` would fail because there
are no further expressions to try.

### Normalization using Template Pipelines

If you have a _very_ unpredictable data structure, it may be better to
normalize it into a more predictable shape as a separate step before running
`hcltemplate`. You can do that using a custom program or using `jq`. You could
also, in principle, use `hcltemplate` itself to do it, by piping its own output
into itself to create a pipeline of templates, as long as all of the
intermediate templates consist only of a call to `jsonencode`:

```
${jsonencode({
  "name": try(name, null),
  "friends": try(
    convert(friends, list(string)),
    [convert(friends, string)],
    [],
  ),
})}
```

If we name the above `normalize.tmpl` and our "main" template is called
`main.tmpl` then we can pipeline this as follows:

```
hcltemplate normalize.tmpl <input.json | hcltemplate main.tmpl
```

The `normalize.tmpl` file ensures that the result will always have a
predictable structure, and so `main.tmpl` can just assume that structure and
thus it would not need to include so many repetitive `try` and `convert`
calls.
