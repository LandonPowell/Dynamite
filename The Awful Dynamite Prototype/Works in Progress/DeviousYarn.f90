subroutine tokenizer(stringInput, tokenList)
    character, intent(in)  :: stringInput        ! Input
    character, intent(out) :: tokenList(2000000) ! Output

    print *, "Doing Function"
end subroutine

program DeviousYarn

    character(2000000) :: input !2MB for input.
    do while (input /= "KILL")

        print *, "-input-"
        read    (*, "(A)") input

        call tokenizer(input, input)

        print *, "-output-"
        write   (*, "(A)") trim( input )

    end do

    print *, "Thanks for using DeviousYarn~!"
end program DeviousYarn
