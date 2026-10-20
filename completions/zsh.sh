#compdef nib
#autoload


_nib() {
    compadd -- $(nib _completions "${words[@]}")
}

compdef _nib nib
