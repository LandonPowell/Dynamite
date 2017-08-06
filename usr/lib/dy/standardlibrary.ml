type value =
    | Number of float
    | String of string
    | Boolean of bool
    | Array of value array
;;

let to_ocaml_string str = match str with
    | String str -> str
    | _ ->
        raise (Invalid_argument "Can't convert non-string to ocaml string.")
;;

let to_ocaml_float num = match num with
    | Number flt -> flt
    | _ ->
        raise (Invalid_argument "Can't convert non-number to ocaml float.")
;;

let to_ocaml_int num = match num with
    | Number flt -> int_of_float flt
    | _ ->
        raise (Invalid_argument "Can't convert non-number to ocaml integer.")
;;

let to_ocaml_bool boolean = match boolean with
    | Boolean b -> b
    | _ ->
        raise (Invalid_argument "Can't convert non-bool to ocaml bool.")
;;

(* Number operations. *)

(*
let plus x y = (to_ocaml_float x) +. (to_ocaml_float y);;
let minus x y = (to_ocaml_float x) -. (to_ocaml_float y);;

let divide x y = (to_ocaml_float x) /. (to_ocaml_float y);;
let multiply x y = (to_ocaml_float x) *. (to_ocaml_float y);;
*)

let exponent x y = Number ((to_ocaml_float x) ** (to_ocaml_float y));;
let divisible x y = Boolean ((to_ocaml_int x) mod (to_ocaml_int y) = 0);;

(* String operations. *)

let print str = print_string (to_ocaml_string str); str;;

let rec out v = match v with
    | Number flt -> (print_float flt; print_string "\n"); v
    | String str -> (print_endline str); v
    | Boolean bool -> (if bool then print_endline "true" else print_endline "false"); v
    | Array values -> (Array.map out values); v
;;

(* Boolean operations. *)

let is x y = Boolean (x = y);;

(* Array operations. *)

let index array i = match array with
    | Array arr -> arr.(
        match i with 
            | Number n -> int_of_float n
            | _ -> raise (Invalid_argument "Can't use non-number as an index.")
    )

    | _ ->
        raise (Invalid_argument "Can't get index from a non-array.")
;;
