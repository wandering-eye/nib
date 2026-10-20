# Nushell completions for nib.
#
# Quick install:
#   nib nushell-completion | save -f ~/.config/nushell/completions/nib.nu
# Then in your config.nu (or any sourced file):
#   source ~/.config/nushell/completions/nib.nu

def "nu-complete nib" [context: string] {
    let raw = ($context | split row " " | where {|x| $x != ""})
    let args = (if ($context | str ends-with " ") {
        $raw | append ""
    } else {
        $raw
    })

    ^nib _completions ...$args | lines | where {|x| $x != ""}
}

export extern "nib" [
    ...args: string@"nu-complete nib"
]
