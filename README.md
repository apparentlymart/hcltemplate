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
$ terraform-config-inspect --json modules/network | hcltemplate terraform-example.tmpl 
The module has the following input variables:
- environment (string)

The module has the following output values:
- vpc_id
```
