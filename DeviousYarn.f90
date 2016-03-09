program DeviousYarn
    character(2000000) :: input
    do while (.NOT. input .EQ. "KILL")

        print*, "-input-"
        read    (*, "(A)") input

        print*, "-output-"
        write   (*, "(A)") trim( input )

    end do
end program DeviousYarn
