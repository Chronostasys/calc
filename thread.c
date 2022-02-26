//Import 
#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>

//global variables
pthread_cond_t      condA  = PTHREAD_COND_INITIALIZER;
pthread_cond_t      condB  = PTHREAD_COND_INITIALIZER;
pthread_mutex_t     mutex = PTHREAD_MUTEX_INITIALIZER;




void *threadA()
{
    int i = 0, rValue, loopNum;

    while(i<5)
    {
        //unlock mutex
        rValue = pthread_mutex_unlock(&mutex);

        //do stuff
        for(loopNum = 1; loopNum <= 5; loopNum++)
            printf("Hello %d\n", loopNum);

        //signal condition of thread b
        rValue = pthread_cond_signal(&condB);

        //lock mutex
        rValue = pthread_mutex_lock(&mutex);

        //wait for turn
        while( pthread_cond_wait(&condA, &mutex) != 0 )

        i++;
    }

}



void *threadB()
{
    int n = 0, rValue;

    while(n<5)
    {
        //lock mutex
        rValue = pthread_mutex_lock(&mutex);

        //wait for turn
        while( pthread_cond_wait(&condB, &mutex) != 0 )

        //unlock mutex
        rValue = pthread_mutex_unlock(&mutex);

        //do stuff
        printf("Goodbye");

        //signal condition a
        rValue = pthread_cond_signal(&condA);

        n++;        
    }
}




int main(int argc, char *argv[])
{
    //create our threads
    pthread_t a, b;

    pthread_create(&a, NULL, threadA, NULL);
    pthread_create(&b, NULL, threadB, NULL);

    pthread_join(a, NULL);
    pthread_join(b,NULL);
}