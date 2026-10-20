# wire bash completions to nib completion engine. Some of the workarounds for
# idiosyncrasies around separation with colons taken from /etc/bash_completion

_nib() {
    # reconstruct COMP_WORDS to re-join separations caused by colon (which is a default separator)
    # yes, this method ends up splitting by spaces, but that's not a problem for the nib parser
    # see http://tiswww.case.edu/php/chet/bash/FAQ
    original_args=( $(echo "${COMP_WORDS[@]}" | sed 's/ : /:/g' | sed 's/ :$/:/g') )

    # hand to nib as canonical args
    COMPREPLY=( $(nib _completions "${original_args[@]}") )

    # convert nib's suggestions to remove prefix before colon so complete can understand it
    local last_arg="${original_args[-1]}"
    local colon_word=${last_arg%"${last_arg##*:}"}
    local i=${#COMPREPLY[*]}
    while [[ $((--i)) -ge 0 ]]; do
        COMPREPLY[$i]=${COMPREPLY[$i]#"$colon_word"}
    done
}

complete -F _nib nib
complete -F _nib task
#complete -F _nib n
#complete -F _nib t
