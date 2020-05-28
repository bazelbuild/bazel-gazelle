# Copyright 2020 The Bazel Authors. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

def _known_imports_impl(ctx):
    args = ctx.actions.args()
    args.add("-proto_csv", ctx.file.src)
    args.add("-known_imports", ctx.outputs.out)
    args.add("-package", ctx.attr.package)
    args.add("-var", ctx.attr.var)
    args.add("-key", str(ctx.attr.key))
    args.add("-value", str(ctx.attr.value))
    ctx.actions.run(
        executable = ctx.executable._bin,
        inputs = [ctx.file.src],
        outputs = [ctx.outputs.out],
        arguments = [args],
        mnemonic = "KnownImports",
    )
    return [DefaultInfo(files = depset([ctx.outputs.out]))]

known_imports = rule(
    implementation = _known_imports_impl,
    attrs = {
        "src": attr.label(
            allow_single_file = True,
            mandatory = True,
        ),
        "out": attr.output(mandatory = True),
        "package": attr.string(mandatory = True),
        "var": attr.string(mandatory = True),
        "key": attr.int(mandatory = True),
        "value": attr.int(mandatory = True),
        "_bin": attr.label(
            default = Label("//language/proto/gen:gen_known_imports"),
            executable = True,
            cfg = "exec",
        ),
    },
)
