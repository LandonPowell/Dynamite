open Parser

let scope = "global";; (* Can be changed from within the program. Decides current scope, obviously. *)

let prefix = ref "
open Standardlibrary
let global : (string, Obj.t) Hashtbl.t = Hashtbl.create 255;;
";; (* Included at the top of generated code. *)

let varTypes = Hashtbl.create 255;; (* Hashtable used for type checking. *)

let rec expressionLoop abstractTrees =
    String.concat " " ( List.map expression abstractTrees )

and expression abstractTree = match abstractTree with
    | String value -> (
        "(String \"" ^ String.escaped value ^ "\")"
    )

    | Number value -> (
        "(Number " ^ string_of_float value ^ ")"
    )

    | Call ("set", (String name)::trees) -> ( (* [expressionLoop trees] might not be a long-term solution. *)
        Hashtbl.replace varTypes (scope ^ "." ^ name) "Typeless";
        "Hashtbl.replace " ^ scope ^ " \"" ^ String.escaped name ^ "\" (Obj.repr (" ^ expressionLoop trees ^ "));\n"
    )

    | Call (value, trees) -> ( (* To-do: Support both default and user defined vars. *)
        if Hashtbl.mem varTypes (scope ^ "." ^ value) then
            "((Obj.obj (Hashtbl.find " ^ scope ^ " \"" ^ String.escaped value ^ "\")) " ^ (
                expressionLoop trees
            ) ^ ")"
        else
            "(" ^ value ^ " " ^ (expressionLoop trees) ^ ")"
    )

    | List trees -> (match trees with
        | Call ("set", args) :: list ->
            "(let hashtable = Hashtbl.create 255 in\n" ^ (String.concat ";\n" (

                List.map (fun element -> match element with
                    | Call ("set", (String name)::trees) ->
                        "Hashtbl.add hashtable \"" ^ name ^ "\" (" ^ expressionLoop trees ^ ")"

                    | _ -> (print_endline "ERROR - MISSING VARIABLE SETTING"; "")

                ) trees
            )) ^ ";\nhashtable)"

        | _ ->
            "(Array [|" ^ (String.concat ";" (
                List.map expression trees
            )) ^ "|])"
    )
;;


let transpile code =
    let trees, leftover = parserLoop (tokenize code) in
    let generatedCode = expressionLoop trees in
    !prefix ^ generatedCode
;;
