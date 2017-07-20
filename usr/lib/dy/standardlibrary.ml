type value =
    | Number of float
    | String of string
    | Array of value array
;;

let index array i = match array with
    | Array arr -> arr.(
        match i with 
            | Number n -> int_of_float n
            | _ -> raise (Invalid_argument "Can't use non-number as an index.")
    )

    | _ ->
        raise (Invalid_argument "Can't get index from a non-array.")
;;

let to_ocaml_string str = match str with
    | String str -> str
    | _ ->
        raise (Invalid_argument "Can't convert non-string to ocaml string.")
;;

let print str = print_endline (to_ocaml_string str);;
