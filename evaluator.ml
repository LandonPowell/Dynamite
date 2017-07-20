(*| 
| | Apply and eval meme.
|*)

let rec evaluateLoop trees =
    if List.length trees = 1 then
        evaluate ( List.hd trees )
    else (
        ignore (evaluate ( List.hd trees )); (* "ignore" function uses the side-effect only. *)
        evaluateLoop ( List.tl trees )
    )

and evaluate tree = match tree with
    | Call (("out" | "o"), value) ->
        String ( printLoop value )

    (* INFINITE LOOP CATCHING *)
    | String value -> String value
    | Number value -> Number value

    (* ERROR CATCHING *)
    | Call (value, listdata) ->
        ( print_endline ( String.concat "" ["An attempt to call '"; value; "' was made, but it's undefined."] );
        Number 0.0)

and printLoop trees =
    if List.length trees = 1 then
        printTree ( List.hd trees )
    else (
        ignore ( printTree ( List.hd trees ) );
        printLoop ( List.tl trees )
    )

and printTree tree = match tree with
    | String value ->
        ( print_string value;
        value)
    | Number value ->
        ( print_float value;
        string_of_float value)
    | Call ( value, listdata ) ->
        printTree ( evaluate ( Call ( value, listdata ) ) )
;;

let trees, leftover = parserLoop ( tokenize "o:o:'lol'" );;
evaluateLoop trees;;