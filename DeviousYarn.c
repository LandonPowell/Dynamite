#include <stdio.h>

int main(int argc, char *argv[]) {
    if(argc == 2) {
        FILE* file = fopen(argv[1], "r");
        if(!file) printf("ERR: %s was not opened. \n", argv[1]);
        else      printf("%s opened successfully! \n", argv[1]);
    }

    char *input;

    printf("?> ");
    scanf("%s",input);
}
