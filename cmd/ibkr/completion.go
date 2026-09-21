// Copyright 2026 shing1211
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
)

func runCompletion() error {
	args := os.Args[2:]
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: ibkr completion <bash|zsh|fish>\n")
		return nil
	}

	shell := args[0]
	switch shell {
	case "bash":
		printBashCompletion()
	case "zsh":
		printZshCompletion()
	case "fish":
		printFishCompletion()
	default:
		return fmt.Errorf("unsupported shell %q (want bash, zsh, or fish)", shell)
	}
	return nil
}

func printBashCompletion() {
	fmt.Println(`# ibkr bash completion
_ibkr() {
    local cur prev commands subcommands
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    commands="accounts positions orders stream portfolio config completion version help"

    if [[ ${cur} == -* ]]; then
        COMPREPLY=( $(compgen -W "-gateway -rest -account -insecure -h -v" -- "${cur}") )
        return 0
    fi

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
        return 0
    fi

    case "${COMP_WORDS[1]}" in
        orders)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "list submit cancel" -- "${cur}") )
            elif [[ "${COMP_WORDS[2]}" == "submit" ]]; then
                COMPREPLY=( $(compgen -W "-conid -side -qty -type -price -stop -tif -rth -coid" -- "${cur}") )
            elif [[ "${COMP_WORDS[2]}" == "cancel" ]]; then
                COMPREPLY=( $(compgen -W "-orderid" -- "${cur}") )
            fi
            ;;
        portfolio)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "summary ledger allocation" -- "${cur}") )
            fi
            ;;
        config)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "show set" -- "${cur}") )
            fi
            ;;
        stream)
            COMPREPLY=( $(compgen -W "-conid -fields" -- "${cur}") )
            ;;
        positions)
            COMPREPLY=( $(compgen -W "-account" -- "${cur}") )
            ;;
    esac
    return 0
}
complete -F _ibkr ibkr`)
}

func printZshCompletion() {
	fmt.Println(`#compdef ibkr

_ibkr() {
    local -a commands
    commands=(
        'accounts:List brokerage accounts'
        'positions:List positions for an account'
        'orders:Manage orders (list, submit, cancel)'
        'stream:Subscribe to market data'
        'portfolio:Portfolio operations (summary, ledger, allocation)'
        'config:Configuration management'
        'completion:Generate shell completions'
        'version:Show version'
        'help:Show help'
    )

    _arguments -C \
        '(-g --gateway)'{-g,--gateway}'[Client Portal Gateway URL]:url' \
        '(-r --rest)'{-r,--rest}'[REST gateway URL]:url' \
        '(-a --account)'{-a,--account}'[Account ID]:account' \
        '(-i --insecure)'{-i,--insecure}'[Skip TLS verification]' \
        '(-h --help)'{-h,--help}'[Show help]' \
        '(-v --version)'{-v,--version}'[Show version]' \
        '1:command:->command' \
        '*::arg:->args'

    case "$state" in
        command)
            _describe 'command' commands
            ;;
        args)
            case "${words[1]}" in
                orders)
                    _arguments -C \
                        '1:subcommand:(list submit cancel)' \
                        '*::arg:->orders_args'
                    case "$state" in
                        orders_args)
                            case "${words[1]}" in
                                submit)
                                    _arguments \
                                        '(-conid)'{-conid}'[Contract ID]:conid' \
                                        '(-side)'{-side}'[BUY or SELL]:side' \
                                        '(-qty)'{-qty}'[Order quantity]:qty' \
                                        '(-type)'{-type}'[Order type]:type' \
                                        '(-price)'{-price}'[Limit price]:price' \
                                        '(-stop)'{-stop}'[Stop price]:price' \
                                        '(-tif)'{-tif}'[Time-in-force]:tif' \
                                        '(-rth)'{-rth}'[Outside RTH]' \
                                        '(-coid)'{-coid}'[Client order ID]:id'
                                    ;;
                                cancel)
                                    _arguments \
                                        '(-orderid)'{-orderid}'[Order ID]:orderid'
                                    ;;
                            esac
                            ;;
                    esac
                    ;;
                portfolio)
                    _arguments -C \
                        '1:subcommand:(summary ledger allocation)'
                    ;;
                config)
                    _arguments -C \
                        '1:subcommand:(show set)'
                    ;;
                stream)
                    _arguments \
                        '(-conid)'{-conid}'[Contract ID]:conid' \
                        '(-fields)'{-fields}'[Field list]:fields'
                    ;;
                positions)
                    _arguments \
                        '(-account)'{-account}'[Account ID]:account'
                    ;;
            esac
            ;;
    esac
}

_ibkr "$@"`)
}

func printFishCompletion() {
	fmt.Println(`# ibkr fish completion

function __ibkr_commands
    echo -e "accounts\tList brokerage accounts"
    echo -e "positions\tList positions"
    echo -e "orders\tManage orders"
    echo -e "stream\tSubscribe to market data"
    echo -e "portfolio\tPortfolio operations"
    echo -e "config\tConfiguration management"
    echo -e "completion\tGenerate shell completions"
    echo -e "version\tShow version"
    echo -e "help\tShow help"
end

function __ibkr_order_subcommands
    echo -e "list\tList open orders"
    echo -e "submit\tSubmit a new order"
    echo -e "cancel\tCancel an open order"
end

function __ibkr_portfolio_subcommands
    echo -e "summary\tPortfolio summary"
    echo -e "ledger\tAccount ledger"
    echo -e "allocation\tAsset allocation"
end

function __ibkr_config_subcommands
    echo -e "show\tShow configuration"
    echo -e "set\tSet a value"
end

complete -c ibkr -f
complete -c ibkr -n '__fish_use_subcommand' -a '(__ibkr_commands)' -d 'Command'
complete -c ibkr -n '__fish_use_subcommand' -s g -l gateway -r -d 'Gateway URL'
complete -c ibkr -n '__fish_use_subcommand' -s r -l rest -r -d 'REST gateway URL'
complete -c ibkr -n '__fish_use_subcommand' -s a -l account -r -d 'Account ID'
complete -c ibkr -n '__fish_use_subcommand' -s i -l insecure -d 'Skip TLS verification'
complete -c ibkr -n '__fish_use_subcommand' -s h -l help -d 'Show help'
complete -c ibkr -n '__fish_use_subcommand' -s v -l version -d 'Show version'
complete -c ibkr -n '__fish_seen_subcommand_from orders' -a '(__ibkr_order_subcommands)'
complete -c ibkr -n '__fish_seen_subcommand_from portfolio' -a '(__ibkr_portfolio_subcommands)'
complete -c ibkr -n '__fish_seen_subcommand_from config' -a '(__ibkr_config_subcommands)'
complete -c ibkr -n '__fish_seen_subcommand_from orders; and __fish_seen_subcommand_from submit' -l conid -r -d 'Contract ID'
complete -c ibkr -n '__fish_seen_subcommand_from orders; and __fish_seen_subcommand_from submit' -l side -r -d 'BUY or SELL'
complete -c ibkr -n '__fish_seen_subcommand_from orders; and __fish_seen_subcommand_from submit' -l qty -r -d 'Quantity'
complete -c ibkr -n '__fish_seen_subcommand_from orders; and __fish_seen_subcommand_from submit' -l type -r -d 'Order type'
complete -c ibkr -n '__fish_seen_subcommand_from orders; and __fish_seen_subcommand_from submit' -l price -r -d 'Limit price'
complete -c ibkr -n '__fish_seen_subcommand_from orders; and __fish_seen_subcommand_from submit' -l stop -r -d 'Stop price'
complete -c ibkr -n '__fish_seen_subcommand_from orders; and __fish_seen_subcommand_from submit' -l tif -r -d 'Time-in-force'
complete -c ibkr -n '__fish_seen_subcommand_from orders; and __fish_seen_subcommand_from submit' -l rth -d 'Outside RTH'
complete -c ibkr -n '__fish_seen_subcommand_from orders; and __fish_seen_subcommand_from submit' -l coid -r -d 'Client order ID'
complete -c ibkr -n '__fish_seen_subcommand_from orders; and __fish_seen_subcommand_from cancel' -l orderid -r -d 'Order ID'
complete -c ibkr -n '__fish_seen_subcommand_from stream' -l conid -r -d 'Contract ID'
complete -c ibkr -n '__fish_seen_subcommand_from stream' -l fields -r -d 'Field list'
complete -c ibkr -n '__fish_seen_subcommand_from positions' -l account -r -d 'Account ID'`)
}
