compdef _zetgrep zetgrep

function _zetgrep {
    _arguments "1: :($(zetgrep -list))"
}
