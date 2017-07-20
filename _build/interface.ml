(* To-do : Make these lines less ugly *)
let sourceFile = if (Array.length Sys.argv) < 2 then "main" else Sys.argv.(1);;
let nativeFile = if (Array.length Sys.argv) < 3 then "executable" else Sys.argv.(2);;
let outputFile = if (Array.length Sys.argv) < 4 then sourceFile ^ ".ml" else Sys.argv.(3);;

let sourceCode =
    let file = open_in (sourceFile ^ ".dy") in
    let source = ref "" in
    try while true; do
        source := !source ^ (String.make 1 (input_char file))
    done; !source
    with e ->
        close_in file;
        !source
;;

let ocamlCode = Codegen.transpile sourceCode;;

let () =
    (try Unix.mkdir "./ocamlCode" 0o777 with e -> ());
    let file = open_out ("./ocamlCode/" ^ outputFile) in
    Printf.fprintf file "%s" ocamlCode;
    close_out file;
    Unix.chdir "./ocamlCode";
    (* To-do : Move the standard library into the ocamlCode folder. *)
    ignore( Unix.system ("ocamlbuild -use-ocamlfind " ^ sourceFile ^ ".native ") );
    ignore( Unix.system ("mv " ^ sourceFile ^ ".native ../" ^ nativeFile) );
;;
