function nib () {
    local has_double_dash subcommand task_ids

    if (( $# == 1 || $# == 2 )); then
        has_double_dash="$( [[ "$1" == "--" || "$2" == "--" ]] && echo "1" || echo "0" )"
        subcommand="$( [[ "$1" == "--" ]] && echo "$2" || echo "$1" )"
        if (( $# == 1 || has_double_dash )) && { \
            [[ "${subcommand}" =~ ^(done|edit|note|open|remove|start|stop)$ ]] || \
            [[ "${subcommand}" == "modify" && -n "${WIDGET}" ]] \
        }; then
            task_ids=("${(@f)$( \
                command nib show-open --json $( (( has_double_dash )) && echo "--" ) \
                | jq -r '.[] | [.uuid, .summary] | @tsv' \
                | fzf --multi --delimiter='\t' --nth=2 --accept-nth=1 --preview=' nib {1} --' \
            )}")
            (( ${#task_ids[@]} > 0 )) || return
            if [[ "${subcommand}" == "modify" ]]; then
                RBUFFER=""
                LBUFFER="nib modify ${task_ids[@]}"
            else
                xargs -n1 nib "${subcommand}" <<< "${task_ids[@]}"
            fi
            return
        fi
    fi

    command nib "$@"
}

function nib-accept-line () {
    if [[ "${BUFFER}" =~ "^\s*nib\s+modify\s*$" ]]; then
        nib modify
        zle redisplay
    else
        zle orig-accept-line
    fi
}
zle -A accept-line orig-accept-line
zle -N accept-line nib-accept-line
