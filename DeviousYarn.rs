use std::io;

fn main() {
    while(true) {
        println!(" -input- ");
        let mut input = String::new();
        io::stdin().read_line(&mut input);
        println!(" -output- \n{}", input);
    }
    println!(" Thanks for using DeviousYarn~! ")
}
