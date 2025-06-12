package main

import (
	"fmt"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic" // Import the gamelogic package for game state management
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"    // Import the pubsub package for message handling
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"   // Import the routing package for routing keys
	amqp "github.com/rabbitmq/amqp091-go"
)

// handlerMove processes army moves and handles the outcomes of those moves
func handlerMove(gs *gamelogic.GameState, publishCh *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(move gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")

		moveOutcome := gs.HandleMove(move)
		switch moveOutcome {
		case gamelogic.MoveOutcomeSamePlayer:
			return pubsub.NackDiscard
		case gamelogic.MoveOutComeSafe:
			return pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			err := pubsub.PublishJSON(
				publishCh,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.GetUsername(),
				gamelogic.RecognitionOfWar{
					Attacker: move.Player,
					Defender: gs.GetPlayerSnap(),
				},
			)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		}

		fmt.Println("error: unknown move outcome")
		return pubsub.NackDiscard
	}
}

// handlerWar processes war declarations and determines the outcome of the war
func handlerWar(gs *gamelogic.GameState, publishCh *amqp.Channel) func(dw gamelogic.RecognitionOfWar) pubsub.Acktype {
	// Handle the war declaration and determine the outcome
	return func(dw gamelogic.RecognitionOfWar) pubsub.Acktype {
		// Print a prompt after handling the war declaration
		defer fmt.Print("> ")
		// Handle the war declaration and get the outcome
		warOutcome, winner, loser := gs.HandleWar(dw)
		switch warOutcome {
		case gamelogic.WarOutcomeNotInvolved:
			return pubsub.NackRequeue // Requeue the message for later processing
		case gamelogic.WarOutcomeNoUnits:
			return pubsub.NackDiscard // Discard the message if there are no units involved in the war
		case gamelogic.WarOutcomeOpponentWon:
			err := publishGameLog(
				publishCh,
				gs.GetUsername(),
				fmt.Sprintf("%s won a war against %s", winner, loser),
			)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack // Acknowledge the message if the opponent won
		case gamelogic.WarOutcomeYouWon:
			err := publishGameLog(
				publishCh,
				gs.GetUsername(),
				fmt.Sprintf("%s won a war against %s", winner, loser),
			)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack // Acknowledge the message if you won the war
		case gamelogic.WarOutcomeDraw:
			err := publishGameLog(
				publishCh,
				gs.GetUsername(),
				fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser),
			)
			if err != nil {
				fmt.Printf("error: %s\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack // Acknowledge the message if the war ended in a draw
		}

		fmt.Println("error: unknown war outcome")
		return pubsub.NackDiscard
	}
}

// handlerPause processes pause requests and updates the game state accordingly
func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	// Handle the pause request and update the game state accordingly
	return func(ps routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(ps)
		return pubsub.Ack
	}
}
